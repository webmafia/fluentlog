package msgpack

import "io"

var _ io.Reader = binReader{}

type binReader struct {
	iter *Iterator
}

// Read implements io.Reader.
func (b binReader) Read(p []byte) (n int, err error) {
	if b.iter.remain < len(p) {
		p = p[:b.iter.remain]
	}

	// If there's nothing left to read, return EOF
	if b.iter.remain == 0 {
		return 0, io.EOF
	}

	// Read from both the buffer and the io.Reader
	for n < len(p) {

		// Determine how many bytes can be read
		bufLen := len(b.iter.buf)
		unreadLen := bufLen - b.iter.n

		// First, read from the iterator's buffer
		if unreadLen > 0 {
			copied := copy(p[n:], b.iter.buf[b.iter.n:])
			// b.iter.buf = b.iter.buf[:bufLen-copied]
			b.iter.n += copied
			n += copied
		} else {

			// Then, read the rest directly from the io.Reader
			readBytes, readErr := b.iter.r.Read(p[n:])
			n += readBytes

			if readErr != nil {
				err = readErr
				break
			}
		}
	}

	// Ensure EOF is returned when remaining bytes reach 0
	if b.iter.remain == 0 && err == nil {
		err = io.EOF
	}

	b.iter.tot += n
	b.iter.remain -= n

	return
}
