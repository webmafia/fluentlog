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

	// Read from both the buffer and the io.Reader
	for n < len(p) && b.iter.remain > 0 {
		// Determine how many bytes can be read
		toCopy := len(p) - n
		available := len(b.iter.buf) - b.iter.n

		// First, read from the buffer
		if available > 0 {
			if toCopy > available {
				toCopy = available
			}
			if toCopy > b.iter.remain {
				toCopy = b.iter.remain
			}

			copy(p[n:], b.iter.buf[b.iter.n:b.iter.n+toCopy])
			b.iter.n += toCopy
			b.iter.remain -= toCopy
			n += toCopy
		}

		// Then, read from the io.Reader if needed
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
				break
			}
		}
	}

	// Ensure EOF is returned when remaining bytes reach 0
	if b.iter.remain == 0 && err == nil {
		err = io.EOF
	}

	b.iter.tot += n
	return n, err
}
