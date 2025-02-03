package gzip

import (
	"bufio"
	"compress/flate"
	"io"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

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
}

// NewReader creates a new Reader reading the given reader.
// If r does not also implement io.ByteReader,
// the decompressor may read more data than necessary from r.
//
// It is the caller's responsibility to call Close on the Reader when done.
//
// The Reader.Header fields will be valid in the Reader returned.
func NewReader(r msgpack.BinReader) (*Reader, error) {
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
