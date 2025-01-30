package msgpack

import "io"

type BinReader interface {
	io.Reader
	io.ByteReader
}

// Get a reader of the value. The reader MUST be consumed until EOF.
// This method is primarily for Bin data, but works for any other data type
// (e.g. strings) as well.
func (iter *Iterator) BinReader() BinReader {
	iter.remain = iter.Len()
	return binReader{iter: iter}
}

type binReader struct {
	iter *Iterator
}

// Read implements io.Reader.
func (b binReader) Read(p []byte) (n int, err error) {
	if b.iter.remain == 0 {
		return 0, io.EOF
	}

	if b.iter.remain < len(p) {
		p = p[:b.iter.remain]
	}

	n, err = b.readFromBuf(p)

	if n == 0 {
		n, err = b.iter.r.Read(p)
	}

	b.iter.tot += n
	b.iter.remain -= n

	if b.iter.remain == 0 {
		err = io.EOF
		b.iter.n = min(b.iter.t2, len(b.iter.buf))
	} else if err == io.EOF {
		err = io.ErrUnexpectedEOF
		b.iter.n = min(b.iter.t2, len(b.iter.buf))
		b.iter.remain = 0
	}

	return
}

// ReadByte implements io.ByteReader.
func (b binReader) ReadByte() (byte, error) {
	if b.iter.remain == 0 {
		return 0, io.EOF
	}

	// If buffer is empty, refill before reading
	if b.iter.n >= len(b.iter.buf) {
		if err := b.refill(); err != nil {
			return 0, err
		}
	}

	// Read from buffer
	c := b.iter.buf[b.iter.n]
	b.iter.n++
	b.iter.remain--
	b.iter.tot++
	return c, nil
}

// refill fills `iter.buf[iter.t1:]` from `iter.r` while respecting `iter.remain`.
func (b binReader) refill() error {
	if b.iter.remain == 0 {
		return io.EOF
	}

	// Shift unread bytes to the beginning (if necessary)
	unreadLen := len(b.iter.buf) - b.iter.n
	if unreadLen > 0 {
		copy(b.iter.buf[b.iter.t1:], b.iter.buf[b.iter.n:])
	}

	// Determine how much we can safely read (max `iter.remain` bytes)
	toRead := min(cap(b.iter.buf)-b.iter.t1, b.iter.remain)

	// Read into `iter.buf[iter.t1:]`
	n, err := b.iter.r.Read(b.iter.buf[b.iter.t1 : b.iter.t1+toRead])

	// Update buffer pointers
	if n > 0 {
		b.iter.n = b.iter.t1
		b.iter.buf = b.iter.buf[:b.iter.t1+n]
	}

	return err
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

func (b binReader) SkipBytes(n int) error {
	// Ensure we do not exceed remaining bytes
	if n > b.iter.remain {
		b.iter.n = min(b.iter.t2, len(b.iter.buf)) // Ensure correct state
		b.iter.remain = 0
		return io.EOF
	}

	// Use `iter.skipBytes()` to efficiently move forward
	if !b.iter.skipBytes(n) {
		b.iter.n = min(b.iter.t2, len(b.iter.buf))
		b.iter.remain = 0
		return io.ErrUnexpectedEOF
	}

	// Update internal counters
	b.iter.remain -= n

	return nil
}
