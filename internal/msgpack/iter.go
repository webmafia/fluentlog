package msgpack

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type Iterator struct {
	buf []byte    // Buffer
	r   io.Reader // Origin
	t   int       // Token start
	n   int       // Cursor position
	tot int       // Total read bytes
	max int       // Max size of buffer
	rp  int       // Release point
	err error
}

func NewIterator(r io.Reader, maxBufSize ...int) Iterator {
	iter := Iterator{
		r:   r,
		max: 4096,
	}

	if len(maxBufSize) > 0 {
		iter.max = maxBufSize[0]
	}

	return iter
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
	iter.buf = iter.buf[:0]
	iter.n = 0
	iter.t = 0
}

// Read next token. Must be called before any Read* method.
func (iter *Iterator) Next() bool {
	iter.t = iter.n

	if !iter.fill(1) {
		return false
	}

	typ, length, isValueLength := types.Get(iter.buf[iter.t])
	iter.consume(1)

	if !isValueLength {
		if !iter.fill(length) {
			return false
		}

		iter.consume(length)
		length = int(uintFromBuf[uint](iter.buf[iter.t+1 : iter.n]))
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
	typ, _, _ := types.Get(iter.buf[iter.t])
	return typ
}

func (iter *Iterator) IsCollection() bool {
	typ := iter.Type()
	return typ == types.Array || typ == types.Map
}

func (iter *Iterator) Len() int {
	_, length, isValueLength := types.Get(iter.buf[iter.t])

	if !isValueLength {
		length = int(uintFromBuf[uint](iter.buf[iter.t+1 : iter.n]))
	}

	return length
}

func (iter *Iterator) Bin() []byte {
	v, _, _ := ReadBinary(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Str() string {
	v, _, _ := ReadString(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Bool() bool {
	v, _, _ := ReadBool(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Float() float64 {
	v, _, _ := ReadFloat(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Int() int64 {
	v, _, _ := ReadInt(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Uint() uint64 {
	v, _, _ := ReadUint(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Time() time.Time {
	v, _, _ := ReadTimestamp(iter.buf, iter.t)
	return v
}

func (iter *Iterator) Value() Value {
	return Value(iter.buf[iter.t:iter.n])
}

func (iter *Iterator) Skip() {
	typ, length, isValueLength := types.Get(iter.buf[iter.t])

	if !isValueLength {
		length = int(uintFromBuf[uint](iter.buf[iter.t+1 : iter.n]))
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
		if !iter.Next() {
			break
		}

		iter.Skip()
	}
}

// Ensures that there is at least n bytes of data in buffer
func (r *Iterator) fill(n int) bool {
	if n == 0 {
		return true
	}

	l := len(r.buf)
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

	readOffset := len(r.buf) // Start reading from the current end of valid data

	if !r.grow(n) {
		return false
	}

	r.buf = r.buf[:cap(r.buf)] // Expand buffer to its full capacity

	var err error

	for n > 0 {
		// Read data from the io.Reader
		var bytesRead int
		bytesRead, err = r.r.Read(r.buf[readOffset:])

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
	r.buf = r.buf[:readOffset]
	return true
}

// grow copies the buffer to a new, larger buffer so that there are at least n
// bytes of capacity beyond len(b.buf).
func (r *Iterator) grow(n int) bool {
	need := len(r.buf) + n

	// There is already enough capacity
	if need <= cap(r.buf) {
		return true
	}

	// A power-of-two value between 64 and `r.max`
	c := min(r.max, max(64, roundPow(need)))

	if c < need {
		return r.reportError("grow", ErrLargeBuffer)
	}

	buf := fast.MakeNoZeroCap(len(r.buf), c)
	copy(buf, r.buf)
	r.buf = buf

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

	log.Printf("--- RELEASING: %d/%d, whereof %d reserved and %d unused\n", len(r.buf), cap(r.buf), r.rp, r.n-r.rp)

	// Move the unread portion (r.b[r.n:]) down to start at r.rp.
	unreadLen := len(r.buf) - r.n
	copy(r.buf[r.rp:], r.buf[r.n:])

	// Adjust the read cursor: it now points to the start of the moved unread data.
	r.n = r.rp

	// Truncate the buffer so that it ends right after the moved unread data.
	r.buf = r.buf[:r.rp+unreadLen]
}

func (r *Iterator) shouldRelease() bool {
	unused := r.n - r.rp
	c := cap(r.buf)

	// Release only if:
	return c >= 4096 && unused > (3*c/4) // Unused data is significant
}

// func (r *Iterator) Peek(n int) (b []byte, err error) {
// 	if err = r.fill(n); err != nil {
// 		return
// 	}

// 	return r.buf[r.n : r.n+n], nil
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
