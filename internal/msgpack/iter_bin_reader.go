package msgpack

import "io"

func (iter *Iterator) BinReader() io.Reader {
	iter.remain = iter.Len()
	return binReader{iter: iter}
}

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

	if b.iter.remain == 0 {
		err = io.EOF
		b.iter.n = b.iter.t2
	} else if err == io.EOF {
		err = io.ErrUnexpectedEOF
		b.iter.n = b.iter.t2
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

		// If we've just consumed *all* buffered bytes...
		if copied == unreadLen {
			// ... then we can discard them and do an implicit release
			b.releaseToT0()
		}
	}

	return
}

// releaseToT0 shifts unread bytes to t0, truncates the buffer,
// and resets n, t1, t2 to t0—without skipping anything from the stream
// or changing iter.rp.
func (b binReader) releaseToT0() {
	unreadLen := len(b.iter.buf) - b.iter.n
	if unreadLen > 0 {
		copy(b.iter.buf[b.iter.t0:], b.iter.buf[b.iter.n:])
	}

	// Now truncate
	newSize := b.iter.t0 + unreadLen
	b.iter.buf = b.iter.buf[:newSize]

	// Reset pointers to t0
	b.iter.n = b.iter.t0
	b.iter.t1 = b.iter.t0
	b.iter.t2 = b.iter.t0
}
