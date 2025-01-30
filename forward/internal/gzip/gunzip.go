// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gzip implements reading and writing of gzip format compressed files,
// as specified in RFC 1952.
package gzip

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/klauspost/compress/flate"
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
	remaining    int    // Number of valid bytes left in buf
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

func (z *Reader) Read(p []byte) (n int, err error) {
	if z.err != nil {
		return 0, z.err
	}

	// 1. Consume any remaining buffered bytes first
	if z.remaining > 0 {
		n = copy(p, z.buf[:z.remaining]) // Copy from buffer to output
		z.remaining -= n                 // Reduce remaining count
		if z.remaining > 0 {
			copy(z.buf[:z.remaining], z.buf[n:]) // Shift remaining bytes
		}
		return n, nil
	}

	// 2. Read from `z.decompressor`
	for n == 0 {
		n, z.err = z.decompressor.Read(p)
		z.digest = crc32.Update(z.digest, crc32.IEEETable, p[:n])
		z.size += uint32(n)

		if z.err != io.EOF {
			return n, z.err
		}

		// 3. Finished file; validate checksum and size
		if err := z.ensureBytes(8); err != nil {
			if err == io.EOF {
				return 0, io.ErrUnexpectedEOF
			}
			z.err = noEOF(err)
			return 0, z.err
		}

		// 4. Check gzip trailer (CRC + ISIZE)
		trailerDigest := le.Uint32(z.buf[:4])
		trailerSize := le.Uint32(z.buf[4:8])
		if trailerDigest != z.digest || trailerSize != z.size {
			z.err = ErrChecksum
			return 0, z.err
		}
		z.digest, z.size = 0, 0
		z.remaining -= 8 // Mark the trailer bytes as consumed

		// 5. Handle multistream gzip files
		if !z.multistream {
			return 0, io.EOF
		}
		z.err = nil

		// 6. Process the next gzip member
		if z.err = z.skipHeader(); z.err != nil {
			return 0, z.err
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

// skipHeader efficiently skips the gzip header while properly updating `z.remaining`.
func (z *Reader) skipHeader() error {
	if err := z.ensureBytes(10); err != nil {
		return err
	}
	// Parse first 10 bytes from z.buf[0..10].
	if z.buf[0] != gzipID1 || z.buf[1] != gzipID2 || z.buf[2] != gzipDeflate {
		return ErrHeader
	}
	flg := z.buf[3]
	// ... (check reserved bits, etc.)

	// Consume them
	z.remaining -= 10
	// Shift them out
	copy(z.buf[:z.remaining], z.buf[10:10+z.remaining])

	// Now parse optional fields from the front each time
	if flg&flagExtra != 0 {
		// ensure at least 2 new bytes are available
		if err := z.ensureBytes(2); err != nil {
			return err
		}
		extraLen := int(le.Uint16(z.buf[:2]))
		z.remaining -= 2
		copy(z.buf[:z.remaining], z.buf[2:2+z.remaining])

		// skip 'extraLen' bytes
		if err := z.skipBytes(extraLen); err != nil {
			return err
		}
	}

	if flg&flagName != 0 {
		if err := z.skipNullTerminated(); err != nil {
			return err
		}
	}
	if flg&flagComment != 0 {
		if err := z.skipNullTerminated(); err != nil {
			return err
		}
	}
	if flg&flagHdrCrc != 0 {
		if err := z.ensureBytes(2); err != nil {
			return err
		}
		z.remaining -= 2
		copy(z.buf[:z.remaining], z.buf[2:2+z.remaining])
	}

	// Initialize or reset the DEFLATE reader
	if z.decompressor == nil {
		z.decompressor = flate.NewReader(z.r)
	} else {
		z.decompressor.(flate.Resetter).Reset(z.r, nil)
	}
	return nil
}

// ensureBytes ensures that `n` bytes are available in `z.buf`, refilling if needed.
func (z *Reader) ensureBytes(n int) error {
	// If we already have enough bytes in the buffer, return immediately.
	if z.remaining >= n {
		return nil
	}

	// Shift existing unread bytes to the start of the buffer
	if z.remaining > 0 {
		copy(z.buf[:z.remaining], z.buf[len(z.buf)-z.remaining:])
	}

	// Read more data into the buffer
	read, err := io.ReadAtLeast(z.r, z.buf[z.remaining:], n-z.remaining)
	if err != nil {
		return err
	}

	// Update remaining bytes count
	z.remaining += read
	return nil
}

// skipBytes skips `n` bytes, refilling the buffer if necessary.
func (z *Reader) skipBytes(n int) error {
	for n > 0 {
		if z.remaining == 0 {
			if err := z.ensureBytes(1); err != nil {
				return err
			}
		}

		toSkip := n
		if toSkip > z.remaining {
			toSkip = z.remaining
		}

		z.remaining -= toSkip
		n -= toSkip
	}

	return nil
}

// skipNullTerminated skips a null-terminated string, refilling the buffer as needed.
func (z *Reader) skipNullTerminated() error {
	for {
		if z.remaining == 0 {
			if err := z.ensureBytes(1); err != nil {
				return err
			}
		}

		// Search for null byte in the buffer
		if idx := bytes.IndexByte(z.buf[:z.remaining], 0); idx != -1 {
			z.remaining -= idx + 1
			return nil
		}

		// Null not found, consume entire buffer
		z.remaining = 0
	}
}
