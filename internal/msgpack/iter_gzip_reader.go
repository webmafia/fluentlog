package msgpack

import (
	"compress/flate"
	"fmt"
	"io"
)

// GzipReader provides a streaming decompression reader for MessagePack bin values compressed with Gzip.
func (iter *Iterator) GzipReader() io.Reader {
	iter.remain = iter.Len()
	return gzipReader{read: binReader{iter: iter}}
}

type gzipReader struct {
	read binReader
}

// Read implements io.Reader.
func (g gzipReader) Read(p []byte) (int, error) {
	// Ensure we process at least one stream
	if g.read.iter.remain == 0 {
		return 0, io.EOF
	}

	// 1. Skip Gzip header
	if err := g.skipGzipHeader(); err != nil {
		return 0, err
	}

	// 2. Create Deflate reader (no allocations)
	deflateReader := flate.NewReader(g.read)

	// 3. Read decompressed data
	n, err := deflateReader.Read(p)

	// 4. If EOF, skip trailer and check for multistream
	if err == io.EOF {
		if err := g.skipGzipTrailer(); err != nil {
			return n, err
		}

		// If multistream, process next stream
		if g.read.iter.remain > 0 {
			return g.Read(p)
		}
	}

	return n, err
}

func (g gzipReader) skipGzipHeader() error {
	// Access the iterator through `binReader`
	iter := g.read.iter

	if iter.remain < 10 {
		return io.ErrUnexpectedEOF
	}

	// Read the fixed Gzip header (10 bytes)
	buf, err := g.read.iter.Peek(10)
	if err != nil {
		return err
	}

	// Validate magic number and compression method
	if buf[0] != 0x1F || buf[1] != 0x8B || buf[2] != 8 {
		return fmt.Errorf("invalid Gzip header")
	}

	flags := buf[3]

	// 🔹 **Skip Fixed 10-Byte Header**
	if err := g.read.SkipBytes(10); err != nil {
		return err
	}

	// 🔹 **Skip Extra Fields (Optional)**
	if flags&4 != 0 {
		if err := g.read.SkipBytes(2); err != nil { // Read extra field length (2 bytes)
			return err
		}
		extraLenBuf, err := g.read.iter.Peek(2)
		if err != nil {
			return err
		}
		extraLen := int(extraLenBuf[0]) | int(extraLenBuf[1])<<8
		if err := g.read.SkipBytes(extraLen); err != nil {
			return err
		}
	}

	// 🔹 **Skip Filename (Optional)**
	if flags&8 != 0 {
		if err := g.skipNullTerminated(); err != nil {
			return err
		}
	}

	// 🔹 **Skip Comment (Optional)**
	if flags&16 != 0 {
		if err := g.skipNullTerminated(); err != nil {
			return err
		}
	}

	// 🔹 **Skip Header CRC (Optional)**
	if flags&2 != 0 {
		if err := g.read.SkipBytes(2); err != nil {
			return err
		}
	}

	// 🔹 **Final sanity check before decompression**
	if iter.remain < 1 {
		return io.ErrUnexpectedEOF
	}

	return nil
}

// skipNullTerminated reads a null-terminated string (Filename or Comment)
func (g gzipReader) skipNullTerminated() error {
	for {
		b, err := g.read.ReadByte() // Use `binReader.ReadByte()`
		if err != nil {
			return err
		}
		if b == 0 {
			break // Stop at null terminator
		}
	}
	return nil
}

// skipGzipTrailer skips the Gzip checksum and size.
func (g gzipReader) skipGzipTrailer() error {
	if g.read.iter.remain < 8 {
		return io.ErrUnexpectedEOF
	}
	return g.read.SkipBytes(8)
}
