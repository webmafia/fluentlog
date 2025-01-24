package msgpack

import (
	"encoding/binary"
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

func (r *Iterator) Reset(reader io.Reader) {
	r.b.Reset()
	r.r = reader
	r.n = 0
	r.t = 0
}

// Read next token. Must be called before any Read* method.
func (iter *Iterator) Next() (typ types.Type, length int) {
	iter.t = iter.n

	if !iter.fill(1) {
		return
	}

	typ, length, isValueLength := types.Get(iter.b.B[iter.n])
	iter.consume(1)

	if !isValueLength {
		pos := iter.n

		if !iter.fill(length) {
			return
		}

		iter.consume(length)
		length = int(uintFromBuf[uint](iter.b.B[pos:iter.n]))
	}

	return
}

func (iter *Iterator) ReadBinary() []byte {
	typ, length := iter.typLen()

	if typ == types.Array || typ == types.Map {
		return nil
	}

	if !iter.fill(length) {
		return nil
	}

	return iter.bytes(length)
}

func (iter *Iterator) ReadString() string {
	typ, length := iter.typLen()

	if typ == types.Array || typ == types.Map {
		return ""
	}

	if !iter.fill(length) {
		return ""
	}

	return fast.BytesToString(iter.bytes(length))
}

func (iter *Iterator) bytes(n int) []byte {
	pos := iter.n
	iter.consume(n)

	return iter.b.B[pos : pos+n]
}

func (iter *Iterator) ReadBool() bool {
	return iter.b.B[iter.t] == 0xc3
}

func (iter *Iterator) ReadFloat() float64 {
	_, length := iter.typLen()

	if !iter.fill(length) {
		return 0
	}

	return floatFromBuf[float64](iter.bytes(length))
}

func (iter *Iterator) ReadInt() int {
	c := iter.b.B[iter.t]
	typ, length := iter.typLen()

	if !iter.fill(length) {
		return 0
	}

	switch typ {
	case types.Int:
		if length == 0 && c >= 0xe0 {
			return intFromBuf[int]([]byte{c})
		}

		return intFromBuf[int](iter.bytes(length))

	case types.Uint:
		if length == 0 && c <= 0x7f {
			return intFromBuf[int]([]byte{c})
		}

		return int(uintFromBuf[uint](iter.bytes(length)))
	}

	return 0
}

func (iter *Iterator) ReadUint() uint {
	c := iter.b.B[iter.t]
	typ, length := iter.typLen()

	if !iter.fill(length) {
		return 0
	}

	switch typ {
	case types.Int:
		if length == 0 && c >= 0xe0 {
			return uint(intFromBuf[int]([]byte{c}))
		}

		return uint(intFromBuf[int](iter.bytes(length)))

	case types.Uint:
		if length == 0 && c <= 0x7f {
			return uintFromBuf[uint]([]byte{c})
		}

		return uintFromBuf[uint](iter.bytes(length))
	}

	return 0
}

func (iter *Iterator) ReadTime() time.Time {
	c := iter.b.B[iter.t]
	_, length := iter.typLen()

	if !iter.fill(length) {
		return time.Time{}
	}

	src := iter.bytes(length)

	var offset, s, ns int64

	switch c {

	case 0xd6: // Ts32
		if h := src[offset]; h != msgpackTimestamp {
			iter.reportError("ReadTime", expectedExtType(h, msgpackTimestamp))
			return time.Time{}
		}

		offset++

		s = int64(binary.BigEndian.Uint32(src[offset : offset+4]))

	case 0xd7: // Ts64 or Forward EventTime
		h := src[offset]
		offset++

		if h == msgpackTimestamp {
			// Read the combined 64-bit value
			combined := binary.BigEndian.Uint64(src[offset:])

			// Extract nanoseconds (lower 30 bits)
			ns = int64(combined & 0x3FFFFFFF)

			// Extract seconds (upper 34 bits)
			s = int64(combined >> 30)

		} else if h == fluentdEventTime {
			s = int64(int32(binary.BigEndian.Uint32(src[offset : offset+4])))
			ns = int64(int32(binary.BigEndian.Uint32(src[offset+4 : offset+8])))

		} else {
			iter.reportError("ReadTime", expectedExtType(h, msgpackTimestamp))
			return time.Time{}
		}

	case 0xc7: // ext8
		h := src[offset]
		offset++

		if h == msgpackTimestamp {
			ns = int64(binary.BigEndian.Uint32(src[offset : offset+4]))
			s = int64(binary.BigEndian.Uint64(src[offset+4 : offset+12]))

		} else if h == fluentdEventTime {
			s = int64(int32(binary.BigEndian.Uint32(src[offset : offset+4])))
			ns = int64(int32(binary.BigEndian.Uint32(src[offset+4 : offset+8])))

		} else {
			iter.reportError("ReadTime", expectedExtType(h, msgpackTimestamp))
			return time.Time{}
		}

	default:
		s = int64(iter.ReadInt())
	}

	return time.Unix(s, ns)
}

func (iter *Iterator) Skip() bool {
	typ, length := iter.typLen()

	switch typ {

	case types.Array:
		// Do nothing

	case types.Map:
		length *= 2

	default:
		return iter.skipBytes(length)

	}

	for range length {
		iter.Next()

		if !iter.Skip() {
			return false
		}
	}

	return true
}

func (r *Iterator) skipBytes(n int) bool {
	l := len(r.b.B)
	pos := r.n + n

	if pos < l {
		r.n = pos
	} else {
		r.n = l
		n = pos - l

		if err := skipBytes(r.r, n); err != nil {
			return r.reportError("skipBytes", err)
		}
	}

	return true
}

func (iter *Iterator) typLen() (typ types.Type, length int) {
	typ, length, isValueLength := types.Get(iter.b.B[iter.t])

	if !isValueLength {
		length = int(uintFromBuf[uint](iter.b.B[iter.t+1 : iter.n]))
	}

	// iter.t = iter.n
	return
}

// TODO
// func (iter *Iterator) AppendTo(v []byte) []byte {
// 	typ, length := iter.typLen()

// 	v = append(v, iter.b.B[iter.t:iter.n]...)

// }

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
		r.err = io.ErrUnexpectedEOF
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
