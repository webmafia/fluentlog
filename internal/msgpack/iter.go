package msgpack

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type Iterator struct {
	buf    []byte    // Buffer
	r      io.Reader // Origin
	t0     int       // Token head start
	t1     int       // Token value start
	t2     int       // Token value end
	items  int       // Number of array/map items
	n      int       // Cursor position
	tot    int       // Total read bytes
	remain int       // Remaining bytes to read (only used in BinReader)
	max    int       // Max size of buffer
	rp     int       // Release point
	err    error
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
	iter.t0 = 0
	iter.t1 = 0
	iter.t2 = 0
	iter.remain = 0
}

// Read next token. Must be called before any Read* method.
func (iter *Iterator) Next() bool {
	if !iter.skipBytes(iter.t2 - iter.n) {
		return false
	}

	iter.remain = 0
	iter.t0 = iter.n

	if !iter.fill(1) {
		return false
	}

	typ, length, isValueLength := types.Get(iter.buf[iter.t0])
	iter.consume(1)

	if !isValueLength {
		if !iter.fill(length) {
			return false
		}

		iter.consume(length)
		length = int(uintFromBuf[uint](iter.buf[iter.t0+1 : iter.n]))
	}

	iter.t1 = iter.n

	switch typ {

	case types.Array, types.Map:
		iter.t2 = iter.n
		iter.items = length

	// Ext types have on extra "type" byte right before the data
	case types.Ext:
		iter.t2 = iter.n + length + 1
		iter.items = 0

	default:
		iter.t2 = iter.n + length
		iter.items = 0

	}

	return true
}

func (iter *Iterator) fillNext() bool {
	if iter.n >= iter.t2 {
		return true
	}

	length := iter.Len()

	if !iter.fill(length) {
		return false
	}

	iter.consume(length)
	return true
}

func (iter *Iterator) Type() types.Type {
	typ, _, _ := types.Get(iter.buf[iter.t0])
	return typ
}

func (iter *Iterator) Len() int {
	return iter.t2 - iter.t1
}

func (iter *Iterator) Items() int {
	return iter.items
}

func (iter *Iterator) Bin() []byte {
	if !iter.fillNext() {
		return nil
	}

	return iter.buf[iter.t1:iter.t2]
}

func (iter *Iterator) BinReader() io.Reader {
	iter.remain = iter.Len()
	return binReader{iter: iter}
}

func (iter *Iterator) Str() string {
	if !iter.fillNext() {
		return ""
	}

	return fast.BytesToString(iter.buf[iter.t1:iter.t2])
}

func (iter *Iterator) Bool() bool {
	return readBoolUnsafe(iter.buf[iter.t0])
}

func (iter *Iterator) Float() float64 {
	if !iter.fillNext() {
		return 0
	}

	return floatFromBuf[float64](iter.buf[iter.t1:iter.t2])
}

func (iter *Iterator) Int() int64 {
	if !iter.fillNext() {
		return 0
	}

	return readIntUnsafe[int64](iter.buf[iter.t0], iter.buf[iter.t1:iter.t2])
}

func (iter *Iterator) Uint() uint64 {
	if !iter.fillNext() {
		return 0
	}

	return readIntUnsafe[uint64](iter.buf[iter.t0], iter.buf[iter.t1:iter.t2])
}

func (iter *Iterator) Time() time.Time {
	if !iter.fillNext() {
		return time.Time{}
	}

	return readTimeUnsafe(iter.buf[iter.t0], iter.buf[iter.t1:iter.t2])
}

func (iter *Iterator) Value() Value {
	if !iter.fillNext() {
		return nil
	}

	return Value(iter.buf[iter.t0:iter.t2])
}

func (iter *Iterator) Skip() {
	typ, length, isValueLength := types.Get(iter.buf[iter.t0])

	if !isValueLength {
		length = int(uintFromBuf[uint](iter.buf[iter.t0+1 : iter.n]))
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
	n -= (len(r.buf) - r.n)

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
		return r.reportError("grow", ErrReachedMaxBufferSize)
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

// Get total bytes read
func (r *Iterator) Total() int {
	return r.tot
}

// Sets the release point as current position. Anything before this will be kept after release.
func (r *Iterator) SetReleasePoint() {
	r.rp = r.t0
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

func (iter *Iterator) release() {

	// Ensure we're releasing whole tokens, by skipping to the next token.
	iter.skipBytes(iter.t2 - iter.n)

	// Move all unread bytes back to the release point. Returns number of unread bytes.
	unreadLen := copy(iter.buf[iter.rp:], iter.buf[iter.n:])
	iter.buf = iter.buf[:iter.rp+unreadLen]

	// Adjust cursor and buffer
	iter.n = iter.rp
	iter.t0, iter.t1, iter.t2 = iter.rp, iter.rp, iter.rp
}

func (r *Iterator) shouldRelease() bool {
	unused := r.n - r.rp
	c := cap(r.buf)

	// Release only if:
	return c >= 4096 && unused > (3*c/4) // Unused data is significant
}

func (iter *Iterator) reportError(op string, err any) bool {
	if iter.err == nil {
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

func (iter *Iterator) skipBytes(n int) bool {

	// Nothing to skip
	if n <= 0 {
		return true
	}

	l := len(iter.buf)
	pos := iter.n + n

	if pos < l {
		iter.n = pos
	} else {
		iter.n = l
		n = pos - l

		if err := skipBytes(iter.r, n); err != nil {
			return iter.reportError("skipBytes", err)
		}
	}

	return true
}
