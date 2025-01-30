// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gzip implements reading and writing of gzip format compressed files,
// as specified in RFC 1952.
package gzip

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/klauspost/compress/flate"
	"github.com/webmafia/fast"
)

const (
	gzipID1     = 0x1f
	gzipID2     = 0x8b
	gzipDeflate = 8
	flagText    = 1 << 0
	flagHdrCrc  = 1 << 1
	flagExtra   = 1 << 2
	flagName    = 1 << 3
	flagComment = 1 << 4
)

var (
	// ErrChecksum is returned when reading GZIP data that has an invalid checksum.
	ErrChecksum = gzip.ErrChecksum
	// ErrHeader is returned when reading GZIP data that has an invalid header.
	ErrHeader = gzip.ErrHeader
)

var le = binary.LittleEndian

// noEOF converts io.EOF to io.ErrUnexpectedEOF.
func noEOF(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// A Reader is an io.Reader that can be read to retrieve
// uncompressed data from a gzip-format compressed file.
//
// In general, a gzip file can be a concatenation of gzip files,
// each with its own header. Reads from the Reader
// return the concatenation of the uncompressed data of each.
// Only the first header is recorded in the Reader fields.
//
// Gzip files store a length and checksum of the uncompressed data.
// The Reader will return a ErrChecksum when Read
// reaches the end of the uncompressed data if it does not
// have the expected length or checksum. Clients should treat data
// returned by Read as tentative until they receive the io.EOF
// marking the end of the data.
type Reader struct {
	// Header       // valid after NewReader or Reader.Reset
	r            flate.Reader
	br           *bufio.Reader
	decompressor io.ReadCloser
	digest       uint32 // CRC-32, IEEE polynomial (section 8)
	size         uint32 // Uncompressed size (section 2.3.1)
	buf          [512]byte
	err          error
	multistream  bool
}

// NewReader creates a new Reader reading the given reader.
// If r does not also implement io.ByteReader,
// the decompressor may read more data than necessary from r.
//
// It is the caller's responsibility to call Close on the Reader when done.
//
// The Reader.Header fields will be valid in the Reader returned.
func NewReader(r io.Reader) (*Reader, error) {
	z := new(Reader)
	if err := z.Reset(r); err != nil {
		return nil, err
	}
	return z, nil
}

// Reset discards the Reader z's state and makes it equivalent to the
// result of its original state from NewReader, but reading from r instead.
// This permits reusing a Reader rather than allocating a new one.
func (z *Reader) Reset(r io.Reader) error {
	// *z = Reader{
	// 	decompressor: z.decompressor,
	// 	multistream:  true,
	// 	br:           z.br,
	// }

	if rr, ok := r.(flate.Reader); ok {
		z.r = rr
	} else {
		// Reuse if we can.
		if z.br != nil {
			z.br.Reset(r)
		} else {
			z.br = bufio.NewReader(r)
		}

		z.r = z.br
	}

	if r != nil {
		z.err = z.skipHeader()
	}

	return z.err
}

// Multistream controls whether the reader supports multistream files.
//
// If enabled (the default), the Reader expects the input to be a sequence
// of individually gzipped data streams, each with its own header and
// trailer, ending at EOF. The effect is that the concatenation of a sequence
// of gzipped files is treated as equivalent to the gzip of the concatenation
// of the sequence. This is standard behavior for gzip readers.
//
// Calling Multistream(false) disables this behavior; disabling the behavior
// can be useful when reading file formats that distinguish individual gzip
// data streams or mix gzip data streams with other data streams.
// In this mode, when the Reader reaches the end of the data stream,
// Read returns io.EOF. If the underlying reader implements io.ByteReader,
// it will be left positioned just after the gzip stream.
// To start the next stream, call z.Reset(r) followed by z.Multistream(false).
// If there is no next stream, z.Reset(r) will return io.EOF.
func (z *Reader) Multistream(ok bool) {
	z.multistream = ok
}

func (z *Reader) skipHeader() error {
	// Ensure the underlying reader is initialized
	if z.r == nil {
		return ErrHeader
	}

	// Read fixed 10-byte gzip header
	var buf [10]byte
	if _, err := io.ReadFull(z.r, fast.NoescapeBytes(buf[:])); err != nil {
		return err // Returns EOF if gzip stream is incomplete
	}

	// Validate magic number & compression method
	if buf[0] != gzipID1 || buf[1] != gzipID2 || buf[2] != gzipDeflate {
		return ErrHeader
	}
	flg := buf[3] // Flags byte

	// Skip extra fields, filename, and comment if they exist
	if flg&flagExtra != 0 {
		var extraLen [2]byte
		if _, err := io.ReadFull(z.r, extraLen[:]); err != nil {
			return err
		}
		extraSize := int64(le.Uint16(extraLen[:]))
		var discardBuf [32]byte // Small stack buffer
		for extraSize > 0 {
			n := extraSize
			if n > int64(len(discardBuf)) {
				n = int64(len(discardBuf))
			}
			if _, err := z.r.Read(discardBuf[:n]); err != nil {
				return err
			}
			extraSize -= n
		}
	}

	// Skip null-terminated filename if present
	if flg&flagName != 0 {
		for {
			b, err := z.r.ReadByte()
			if err != nil || b == 0 {
				break
			}
		}
	}

	// Skip null-terminated comment if present
	if flg&flagComment != 0 {
		for {
			b, err := z.r.ReadByte()
			if err != nil || b == 0 {
				break
			}
		}
	}

	// Skip header CRC if present
	if flg&flagHdrCrc != 0 {
		var crc [2]byte
		if _, err := io.ReadFull(z.r, crc[:]); err != nil {
			return err
		}
	}

	z.digest = 0
	if z.decompressor == nil {
		z.decompressor = flate.NewReader(z.r)
	} else {
		z.decompressor.(flate.Resetter).Reset(z.r, nil)
	}

	return nil // Successfully skipped the header
}

// Read implements io.Reader, reading uncompressed bytes from its underlying Reader.
func (z *Reader) Read(p []byte) (n int, err error) {
	if z.err != nil {
		return 0, z.err
	}

	for n == 0 {
		n, z.err = z.decompressor.Read(p)
		z.digest = crc32.Update(z.digest, crc32.IEEETable, p[:n])
		z.size += uint32(n)
		if z.err != io.EOF {
			// In the normal case we return here.
			return n, z.err
		}

		// Finished file; check checksum and size.
		if _, err := io.ReadFull(z.r, z.buf[:8]); err != nil {
			z.err = noEOF(err)
			return n, z.err
		}
		digest := le.Uint32(z.buf[:4])
		size := le.Uint32(z.buf[4:8])
		if digest != z.digest || size != z.size {
			z.err = ErrChecksum
			return n, z.err
		}
		z.digest, z.size = 0, 0

		// File is ok; check if there is another.
		if !z.multistream {
			return n, io.EOF
		}
		z.err = nil // Remove io.EOF

		if z.err = z.skipHeader(); z.err != nil {
			return n, z.err
		}
	}

	return n, nil
}

// Close closes the Reader. It does not close the underlying io.Reader.
// In order for the GZIP checksum to be verified, the reader must be
// fully consumed until the io.EOF.
func (z *Reader) Close() error {
	return z.decompressor.Close()
}
