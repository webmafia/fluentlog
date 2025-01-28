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

	n, err = b.readFromBuf(p)

	if n == 0 {
		n, err = b.iter.r.Read(p)
	}

	b.iter.tot += n
	b.iter.remain -= n

	if err == io.EOF && b.iter.remain != 0 {
		err = io.ErrUnexpectedEOF
	}

	return
}

func (b binReader) readFromBuf(p []byte) (n int, err error) {
	bufLen := len(b.iter.buf)
	unreadLen := bufLen - b.iter.n

	if unreadLen > 0 {
		copied := copy(p, b.iter.buf[b.iter.n:])
		b.iter.n += copied
		n += copied
	}

	return
}
