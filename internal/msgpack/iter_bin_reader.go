package msgpack

import (
	"bytes"
	"errors"
	"io"
)

type BinReader interface {
	io.Reader
	io.ByteReader
	io.Seeker
	SeekByte(c byte) error
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

// Seek implements io.Seeker. Only positive offsets, SeekStart, and SeekCurrent are allowed.
//
// Seek sets the offset for the next Read or Write to offset, interpreted according to whence: [SeekStart] means relative to the start of the file, [SeekCurrent] means relative to the current offset, ~~and [SeekEnd] means relative to the end (for example, offset = -2 specifies the penultimate byte of the file)~~. Seek returns the new offset relative to the start of the file or an error, if any.
func (b binReader) Seek(offset int64, whence int) (newOffset int64, err error) {
	if offset < 0 {
		return 0, ErrInvalidOffset
	}

	switch whence {

	case io.SeekStart:
		offset = min(int64(b.iter.t1)+offset-int64(b.iter.n), int64(len(b.iter.buf)))
		fallthrough

	case io.SeekCurrent:
		err = b.SkipBytes(int(offset))

	case io.SeekEnd:
		err = errors.New("SeekEnd is not allowed")

	}

	newOffset = int64(b.iter.n)
	return
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

// SeekByte finds the next occurence of c, and sets the cursor there for the next Read or ReadByte. If c is not found, the whole io.Reader will be consumed and io.EOF is returned.
func (b binReader) SeekByte(c byte) error {
	for b.iter.remain > 0 {
		// Correctly calculate search window
		haystack := b.iter.buf[b.iter.n:min(b.iter.n+b.iter.remain, len(b.iter.buf))]
		idx := bytes.IndexByte(haystack, c)

		if idx >= 0 {
			// Fix: Move cursor **to** the found byte, not past it
			b.iter.n += idx
			b.iter.remain -= idx
			return nil
		}

		// Fix: Ensure we're subtracting the full searched range
		consumed := len(haystack)
		b.iter.n += consumed
		b.iter.remain -= consumed

		if err := b.refill(); err != nil {
			return err
		}
	}

	return io.EOF
}
