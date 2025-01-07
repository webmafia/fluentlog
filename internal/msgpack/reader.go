package msgpack

import (
	"bytes"
	"io"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type Reader struct {
	b   *buffer.Buffer
	r   io.Reader
	n   int
	max int
}

func NewReader(r io.Reader, buf *buffer.Buffer, maxBuf int) Reader {
	buf.Reset()

	return Reader{
		b:   buf,
		r:   r,
		max: maxBuf,
	}
}

func (r *Reader) Reset(reader io.Reader) {
	r.b.Reset()
	r.r = reader
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
		length = int(uintFromBuf[uint](r.b.B[pos:r.n]))
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

func (r *Reader) ReadStr() (s string, err error) {
	v, err := r.Read()

	if err != nil {
		return
	}

	s, _, err = ReadString(v, 0)
	return
}

func (r *Reader) ReadBin() (b []byte, err error) {
	v, err := r.Read()

	if err != nil {
		return
	}

	b, _, err = ReadBinary(v, 0)
	return
}

func (r *Reader) ReadInt() (i int64, err error) {
	v, err := r.Read()

	if err != nil {
		return
	}

	i, _, err = ReadInt(v, 0)
	return
}

func (r *Reader) ReadUint() (i uint64, err error) {
	v, err := r.Read()

	if err != nil {
		return
	}

	i, _, err = ReadUint(v, 0)
	return
}

func (r *Reader) ReadFloat() (f float64, err error) {
	v, err := r.Read()

	if err != nil {
		return
	}

	f, _, err = ReadFloat(v, 0)
	return
}

func (r *Reader) ReadBool() (b bool, err error) {
	v, err := r.Read()

	if err != nil {
		return
	}

	b, _, err = ReadBool(v, 0)
	return
}

func (r *Reader) ReadHead() (v Value, err error) {
	start := r.n

	if err = r.fill(1); err != nil {
		return
	}

	_, length, isValueLength := types.Get(r.b.B[r.n])
	r.n++

	if !isValueLength {
		if err = r.fill(length); err != nil {
			return
		}

		r.n += length
	}

	v = r.b.B[start:r.n]

	return
}

func (r *Reader) ReadComplete(v Value) (Value, error) {
	l := len(v)

	if l > r.n {
		return v, ErrShortBuffer
	}

	start := r.n - l

	// Ensure that this was the last thing read
	if !bytes.Equal(v, r.b.B[start:r.n]) {
		return v, ErrShortBuffer // Todo: Explicit error?
	}

	count := v.Len()

	if typ := v.Type(); typ == types.Array || typ == types.Map {
		if typ == types.Map {
			count *= 2
		}

		for range count {
			t, err := r.Read()

			if err != nil {
				return v, err
			}

			if typ := t.Type(); typ == types.Array || typ == types.Map {
				if _, err := r.ReadComplete(t); err != nil {
					return v, err
				}
			}
		}
	} else {
		if err := r.fill(count); err != nil {
			return v, err
		}

		r.n += count
	}

	return Value(r.b.B[start:r.n]), nil
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
	if r.r == nil {
		return io.EOF
	}

	readOffset := len(r.b.B) // Start reading from the current end of valid data

	if err = r.grow(n); err != nil {
		return
	}

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

// grow copies the buffer to a new, larger buffer so that there are at least n
// bytes of capacity beyond len(b.buf).
func (r *Reader) grow(n int) (err error) {
	need := len(r.b.B) + n

	// There is already enough capacity
	if need <= cap(r.b.B) {
		return
	}

	// A power-of-two value between 64 and `r.max`
	c := min(r.max, max(64, roundPow(need)))

	if c < need {
		return ErrLargeBuffer
	}

	buf := fast.MakeNoZero(c)[:len(r.b.B)]
	copy(buf, r.b.B)
	r.b.B = buf

	return
}

// Get current read position
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
