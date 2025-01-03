package msgpack

import (
	"io"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type Reader struct {
	b *buffer.Buffer
	r io.Reader
	n int
}

func NewReader(r io.Reader, b *buffer.Buffer) Reader {
	b.Reset()

	return Reader{
		b: b,
		r: r,
	}
}

func (r *Reader) Reset() {
	r.b.Reset()
	r.n = 0
}

func (r *Reader) Read() (v Value, err error) {
	start := r.n

	if err = r.fill(1); err != nil {
		return
	}

	t, length, isValueLength := types.Get(r.b.B[r.n])
	r.n++

	if !isValueLength {
		pos := r.n

		if err = r.fill(length); err != nil {
			return
		}

		r.n += length
		length = intFromBuf[int](r.b.B[pos:r.n])
	}

	if t != types.Array && t != types.Map {
		if err = r.fill(length); err != nil {
			return
		}

		r.n += length
	}

	v = r.b.B[start:r.n]

	return
}

// Ensures that there is at least n bytes of data in buffer
func (r *Reader) fill(n int) (err error) {
	l := len(r.b.B)
	n -= (l - r.n)

	if n <= 0 {
		return
	}

	return r.fillFromReader(n)
}

func (r *Reader) fillFromReader(n int) (err error) {

	readOffset := len(r.b.B) // Start reading from the current end of valid data

	r.b.Grow(n)                // Ensure the buffer has enough capacity
	r.b.B = r.b.B[:cap(r.b.B)] // Expand buffer to its full capacity

	for n > 0 {
		// Read data from the io.Reader
		var bytesRead int
		bytesRead, err = r.r.Read(r.b.B[readOffset:])

		if bytesRead > 0 {
			readOffset += bytesRead
			n -= bytesRead
		}

		if err != nil {
			if err == io.EOF {
				err = nil // EOF is not an error unless n > 0 after loop
			}
			break
		}
	}

	if n > 0 {
		err = io.ErrUnexpectedEOF
	}

	// Adjust buffer size to include only valid data
	r.b.B = r.b.B[:readOffset]
	return
}

func (r *Reader) Pos() int {
	return r.n
}

// Release resets the reader state after processing a message.
// It discards consumed data and prepares for the next message.
func (r *Reader) Release(n int) {
	if n < 0 || n > r.n {
		return
	}

	// Move unread data to position `n`
	copy(r.b.B[n:], r.b.B[r.n:])

	// Adjust `r.n` to reflect the new end of valid data
	r.n = n + (len(r.b.B) - r.n)

	// Resize the buffer to include only valid data
	r.b.B = r.b.B[:r.n]
}
