package msgpack

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/ringbuf"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

// Low-level iteration of a MessagePack stream.
type Iterator struct {
	r      *ringbuf.Reader
	err    error
	length int        // Token value length
	items  int        // Number of array/map items
	byt    byte       // Head byte
	typ    types.Type // Token type
}

func NewIterator(r io.Reader) Iterator {
	return Iterator{
		r: ringbuf.NewReader(r),
	}
}

func (iter *Iterator) Error() error {
	return iter.err
}

func (iter *Iterator) Reset(r io.Reader) {
	iter.r.Reset(r)
}

func (iter *Iterator) ResetBytes(b []byte) {
	iter.r.ResetBytes(b)
}

// Read next token. Must be called before any Read* method.
func (iter *Iterator) Next() bool {
	iter.err = nil

	if iter.byt, iter.err = iter.r.ReadByte(); iter.err != nil {
		return false
	}

	typ, length, isValueLength := types.Get(iter.byt)
	iter.typ = typ

	if !isValueLength {
		buf, err := iter.r.ReadBytes(length)

		if err != nil {
			iter.err = err
			return false
		}

		length = int(uintFromBuf[uint](buf))
	}

	switch typ {

	case types.Array, types.Map:
		iter.length = 0
		iter.items = length

	default:
		iter.length = length
		iter.items = 0

	}

	return true
}

func (iter *Iterator) NextExpectedType(expected ...types.Type) (err error) {
	if !iter.Next() {
		if iter.err != nil {
			return iter.err
		}

		return io.EOF
	}

	for _, t := range expected {
		if t == iter.typ {
			return nil
		}
	}

	return fmt.Errorf("%w: expected any of %v, got %s (%02X)", ErrInvalidHeaderByte, *fast.NoescapeVal(&expected), iter.typ, iter.byt)
}

func (iter *Iterator) Type() types.Type {
	return iter.typ
}

func (iter *Iterator) Len() int {
	return iter.length
}

func (iter *Iterator) Items() int {
	return iter.items
}

// Keeping returned bytes after next call to `Next()` is not safe unless
// the buffer is locked with `Lock`.
func (iter *Iterator) raw() (b []byte, ok bool) {
	b, iter.err = iter.r.ReadBytes(iter.length)
	return b, iter.err == nil
}

func (iter *Iterator) Bin() []byte {
	if b, ok := iter.raw(); ok {
		return b
	}

	return nil
}

func (iter *Iterator) Str() string {
	if b, ok := iter.raw(); ok {
		return fast.BytesToString(b)
	}

	return ""
}

func (iter *Iterator) Bool() bool {

	// Booleans are fully contained in the head byte
	return readBoolUnsafe(iter.byt)
}

func (iter *Iterator) Float() float64 {
	if b, ok := iter.raw(); ok {
		return floatFromBuf[float64](b)
	}

	return 0
}

func (iter *Iterator) Int() int64 {
	if b, ok := iter.raw(); ok {
		return readIntUnsafe[int64](iter.byt, b)
	}

	return 0
}

func (iter *Iterator) Uint() uint64 {
	if b, ok := iter.raw(); ok {
		return readIntUnsafe[uint64](iter.byt, b)
	}

	return 0
}

func (iter *Iterator) Time() time.Time {
	if b, ok := iter.raw(); ok {
		return readTimeUnsafe(iter.byt, b)
	}

	return time.Time{}
}

func (iter *Iterator) Reader() *ringbuf.LimitedReader {
	return iter.r.LimitReader(iter.length)
}

func (iter *Iterator) Skip() {
	items := iter.items

	switch iter.typ {

	case types.Array:
		// Do nothing

	case types.Map:
		items *= 2

	default:
		iter.r.Discard(iter.length)

	}

	for range items {
		if !iter.Next() {
			break
		}

		iter.Skip()
	}
}

// Get total bytes read from underlying io.Reader since last reset.
func (r *Iterator) TotalRead() int {
	return r.r.TotalRead()
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

func (iter *Iterator) Any() any {
	switch iter.typ {

	case types.Bool:
		return iter.Bool()

	case types.Int:
		return iter.Int()

	case types.Uint:
		return iter.Uint()

	case types.Float:
		return iter.Float()

	case types.Str:
		return iter.Str()

	case types.Bin:
		return iter.Bin()

	case types.Ext:
		return iter.Time()

	case types.Array:
		return "Array<" + strconv.Itoa(iter.Items()) + ">"

	case types.Map:
		return "Map<" + strconv.Itoa(iter.Items()) + ">"

	default:
		return nil
	}
}

func (iter *Iterator) DebugDump(w io.Writer) {
	iter.r.DebugDump(w)
}
