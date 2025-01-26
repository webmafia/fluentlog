package msgpack

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"
	"unsafe"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type Iterator struct {
	b   *buffer.Buffer // Buffer
	r   io.Reader      // Origin
	t   int            // Token start
	n   int            // Cursor position
	tot int            // Total read bytes
	max int            // Max size of buffer
	rp  int            // Release point
	err error
}

func NewIterator(r io.Reader, buf *buffer.Buffer, maxBuf int) Iterator {
	buf.Reset()

	return Iterator{
		b:   buf,
		r:   r,
		max: maxBuf,
	}
}

func (iter *Iterator) Error() error {
	return iter.err
}

func (iter *Iterator) Reset(reader io.Reader) {
	iter.r = reader
	iter.reset()
}

func (iter *Iterator) ResetBytes(b []byte) {
	if br, ok := iter.r.(*bytes.Reader); ok {
		br.Reset(b)
	} else {
		iter.r = bytes.NewReader(b)
	}

	iter.reset()
}

func (iter *Iterator) reset() {
	iter.b.Reset()
	iter.n = 0
	iter.t = 0
}

// Read next token. Must be called before any Read* method.
func (iter *Iterator) Next() bool {
	iter.t = iter.n

	if !iter.fill(1) {
		return false
	}

	typ, length, isValueLength := types.Get(iter.b.B[iter.t])
	iter.consume(1)

	if !isValueLength {
		if !iter.fill(length) {
			return false
		}

		iter.consume(length)
		length = int(uintFromBuf[uint](iter.b.B[iter.t+1 : iter.n]))
	}

	if typ != types.Array && typ != types.Map {
		if !iter.fill(length) {
			return false
		}

		iter.consume(length)
	}

	return true
}

func (iter *Iterator) Type() types.Type {
	typ, _, _ := types.Get(iter.b.B[iter.t])
	return typ
}

func (iter *Iterator) IsCollection() bool {
	typ := iter.Type()
	return typ == types.Array || typ == types.Map
}

func (iter *Iterator) Len() int {
	_, length, isValueLength := types.Get(iter.b.B[iter.t])

	if !isValueLength {
		length = int(uintFromBuf[uint](iter.b.B[iter.t+1 : iter.n]))
	}

	return length
}

func (iter *Iterator) Bin() []byte {
	v, _, _ := ReadBinary(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Str() string {
	v, _, _ := ReadString(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Bool() bool {
	v, _, _ := ReadBool(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Float() float64 {
	v, _, _ := ReadFloat(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Int() int64 {
	v, _, _ := ReadInt(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Uint() uint64 {
	v, _, _ := ReadUint(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Time() time.Time {
	v, _, _ := ReadTimestamp(iter.b.B, iter.t)
	return v
}

func (iter *Iterator) Value() Value {
	return Value(iter.b.B[iter.t:iter.n])
}

func (iter *Iterator) Skip() {
	typ, length, isValueLength := types.Get(iter.b.B[iter.t])

	if !isValueLength {
		length = int(uintFromBuf[uint](iter.b.B[iter.t+1 : iter.n]))
	}

	switch typ {

	case types.Array:
		// Do nothing

	case types.Map:
		length *= 2

	default:
		return

	}

	for range length {
		iter.Next()
		iter.Skip()
	}
}

// Ensures that there is at least n bytes of data in buffer
func (r *Iterator) fill(n int) bool {
	if n == 0 {
		return true
	}

	l := len(r.b.B)
	n -= (l - r.n)

	if n <= 0 {
		return true
	}

	return r.fillFromReader(n)
}

func (r *Iterator) fillFromReader(n int) bool {
	if r.r == nil {
		return false
	}

	readOffset := len(r.b.B) // Start reading from the current end of valid data

	if !r.grow(n) {
		return false
	}

	r.b.B = r.b.B[:cap(r.b.B)] // Expand buffer to its full capacity

	var err error

	for n > 0 {
		// Read data from the io.Reader
		var bytesRead int
		bytesRead, err = r.r.Read(r.b.B[readOffset:])

		if bytesRead > 0 {
			readOffset += bytesRead
			n -= bytesRead
		}

		if err != nil {
			if err != io.EOF {
				return r.reportError("fillFromReader", err)
			}

			break
		}
	}

	if n > 0 {
		return false
	}

	// Adjust buffer size to include only valid data
	r.b.B = r.b.B[:readOffset]
	return true
}

// grow copies the buffer to a new, larger buffer so that there are at least n
// bytes of capacity beyond len(b.buf).
func (r *Iterator) grow(n int) bool {
	need := len(r.b.B) + n

	// There is already enough capacity
	if need <= cap(r.b.B) {
		return true
	}

	// A power-of-two value between 64 and `r.max`
	c := min(r.max, max(64, roundPow(need)))

	if c < need {
		return r.reportError("grow", ErrLargeBuffer)
	}

	buf := fast.MakeNoZeroCap(len(r.b.B), c)
	log.Printf("--- GROWING: %d -> %d (%d -> %d)", cap(r.b.B), c, *(*uintptr)(unsafe.Pointer(&r.b.B)), *(*uintptr)(unsafe.Pointer(&buf)))
	copy(buf, r.b.B)
	r.b.B = buf

	return true
}

func (r *Iterator) consume(n int) {
	r.n += n
	r.tot += n
}

// Get current read position
// func (r *Iterator) Pos() int {
// 	return r.n
// }

// Get total bytes read
func (r *Iterator) Total() int {
	return r.tot
}

// Sets the release point as current position. Anything before this will be kept after release.
func (r *Iterator) SetReleasePoint() {
	r.rp = r.n
}

func (r *Iterator) ResetReleasePoint() {
	r.rp = 0
}

// Releases the buffer between the release point and the current position.
func (r *Iterator) Release(force ...bool) {
	if r.shouldRelease() || (len(force) > 0 && force[0]) {
		r.release()
	}
}

func (r *Iterator) release() {

	// If r.rp >= r.n, there's either no gap to release, or
	// it's an invalid state we handle like "nothing to release".
	if r.rp >= r.n {
		return
	}

	log.Printf("--- RELEASING: %d/%d, whereof %d reserved and %d unused\n", len(r.b.B), cap(r.b.B), r.rp, r.n-r.rp)

	// Move the unread portion (r.b[r.n:]) down to start at r.rp.
	unreadLen := len(r.b.B) - r.n
	copy(r.b.B[r.rp:], r.b.B[r.n:])

	// Adjust the read cursor: it now points to the start of the moved unread data.
	r.n = r.rp

	// Truncate the buffer so that it ends right after the moved unread data.
	r.b.B = r.b.B[:r.rp+unreadLen]
}

func (r *Iterator) shouldRelease() bool {
	unused := r.n - r.rp
	c := cap(r.b.B)

	// Release only if:
	return c >= 4096 && unused > (3*c/4) // Unused data is significant
}

// func (r *Iterator) Peek(n int) (b []byte, err error) {
// 	if err = r.fill(n); err != nil {
// 		return
// 	}

// 	return r.b.B[r.n : r.n+n], nil
// }

func (iter *Iterator) reportError(op string, err any) bool {
	if iter.err != nil {
		switch v := err.(type) {
		case error:
			if err != io.EOF {
				iter.err = fmt.Errorf("%s: %w", op, v)
			}
		case string:
			iter.err = fmt.Errorf("%s: %s", op, v)
		default:
			iter.err = fmt.Errorf("%s: %v", op, v)
		}
	}

	return false
}
