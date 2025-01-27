package msgpack

import "io"

var _ io.Reader = binReader{}

type binReader struct {
	iter *Iterator
}

// Read implements io.Reader.
func (b binReader) Read(p []byte) (n int, err error) {
	// If there's nothing left to read, return EOF
	if b.iter.remain == 0 {
		return 0, io.EOF
	}

	// Read from the existing buffer first
	for b.iter.n < len(b.iter.buf) && n < len(p) && b.iter.remain > 0 {
		toCopy := len(p) - n
		buffered := len(b.iter.buf) - b.iter.n

		if toCopy > buffered {
			toCopy = buffered
		}
		if toCopy > b.iter.remain {
			toCopy = b.iter.remain
		}

		copy(p[n:], b.iter.buf[b.iter.n:b.iter.n+toCopy])
		b.iter.consume(toCopy)
		b.iter.remain -= toCopy
		n += toCopy
		b.iter.n = n
	}

	// If there's still space in `p` and more data left to read, read from the `io.Reader`
	if n < len(p) && b.iter.remain > 0 {
		toRead := len(p) - n
		if toRead > b.iter.remain {
			toRead = b.iter.remain
		}

		readBytes, readErr := b.iter.r.Read(p[n : n+toRead])
		n += readBytes
		b.iter.remain -= readBytes

		if readErr != nil {
			if readErr == io.EOF && b.iter.remain == 0 {
				err = io.EOF
			} else {
				err = readErr
			}
		}
	}

	// If we finished reading the remaining bytes, return EOF
	if b.iter.remain == 0 && err == nil {
		err = io.EOF
	}

	b.iter.tot += n
	return n, err
}
