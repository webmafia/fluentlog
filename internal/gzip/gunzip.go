// Package gzip implements reading of gzip format compressed files,
// as specified in RFC 1952. This implementation is lazy and skips
// header and trailer (including checksum). For this reason it is
// recommended that you have any other mechanism for framing the bytes.
package gzip

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"

	"github.com/klauspost/compress/flate"
	"github.com/webmafia/fast/bufio"
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

type inflate interface {
	flate.Resetter
	io.ReadCloser
}

// A Reader is an io.Reader that can be read to retrieve
// uncompressed data from a gzip-format compressed file.
//
// In general, a gzip file can be a concatenation of gzip files,
// each with its own header. Reads from the Reader
// return the concatenation of the uncompressed data of each.
type Reader struct {
	br           bufio.BufioReader
	decompressor inflate
}

// NewReader creates a new Reader reading the given reader.
// If r does not also implement io.ByteReader,
// the decompressor may read more data than necessary from r.
//
// It is the caller's responsibility to call Close on the Reader when done.
func NewReader(br bufio.BufioReader) (*Reader, error) {
	z := new(Reader)

	if err := z.Reset(br); err != nil {
		return nil, err
	}

	return z, nil
}

// Reset discards the Reader z's state and makes it equivalent to the
// result of its original state from NewReader, but reading from r instead.
// This permits reusing a Reader rather than allocating a new one.
func (z *Reader) Reset(br bufio.BufioReader) error {
	z.br = br

	if br != nil {
		return z.skipHeader()
	}

	return nil
}

func (z *Reader) Read(p []byte) (n int, err error) {
	for n == 0 {
		// The header was already skipped on reset

		// Decompress (deflate)
		if n, err = z.decompressor.Read(p); err != io.EOF {
			return
		}

		// Skip trailer (checksum + size)
		if _, err = z.br.Discard(8); err != nil {
			return
		}

		// Process the next gzip member by skipping its header
		if err = z.skipHeader(); err != nil {
			return
		}
	}

	return n, nil
}

// Close closes the Reader. It does not close the underlying io.Reader.
func (z *Reader) Close() error {
	return z.decompressor.Close()
}

// skipHeader efficiently skips the gzip header.
func (z *Reader) skipHeader() (err error) {
	var buf []byte

	if buf, err = z.br.ReadBytes(10); err != nil {
		return
	}

	// Parse first 10 bytes from z.buf[0..10].
	if buf[0] != gzipID1 || buf[1] != gzipID2 || buf[2] != gzipDeflate {
		return ErrHeader
	}

	flg := buf[3]

	if flg&flagExtra != 0 {
		if buf, err = z.br.ReadBytes(2); err != nil {
			return
		}

		extraLen := int(le.Uint16(buf))

		if err = z.skipBytes(extraLen); err != nil {
			return
		}
	}

	if flg&flagName != 0 {
		if err = z.skipNullTerminated(); err != nil {
			return
		}
	}

	if flg&flagComment != 0 {
		if err = z.skipNullTerminated(); err != nil {
			return
		}
	}

	if flg&flagHdrCrc != 0 {
		if err = z.skipBytes(2); err != nil {
			return
		}
	}

	// Initialize or reset the DEFLATE reader
	if z.decompressor == nil {
		if dec, ok := flate.NewReader(z.br).(inflate); ok {
			z.decompressor = dec
		} else {
			return errors.New("gzip: failed to init decompressor")
		}
	} else {
		err = z.decompressor.Reset(z.br, nil)
	}

	return
}

// skipBytes skips `n` bytes.
func (z *Reader) skipBytes(n int) (err error) {
	_, err = z.br.Discard(n)
	return
}

// skipNullTerminated skips a null-terminated string.
func (z *Reader) skipNullTerminated() (err error) {

	// Discard all bytes up to (but not including) null
	if _, err = z.br.DiscardUntil(0); err != nil {
		return
	}

	// Discard the null byte
	_, err = z.br.Discard(1)
	return
}
