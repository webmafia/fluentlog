package fluentlog

import (
	"iter"
	"strconv"
	"time"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Type uint8

const (
	Nil Type = iota
	Bool
	Int
	Uint
	Float
	Str
	Bin
	Array
	Map
	Time
)

func (t Type) String() string {
	switch t {

	case Bool:
		return "bool"

	case Int:
		return "int"

	case Uint:
		return "uint"

	case Float:
		return "float"

	case Str:
		return "str"

	case Bin:
		return "bin"

	case Array:
		return "array"

	case Map:
		return "map"

	case Time:
		return "time"
	}

	return "nil"
}

type Value []byte

func (v Value) Type() Type {
	if len(v) == 0 {
		return Nil // or consider returning an error
	}

	b := v[0]

	// Positive FixInt (0x00 - 0x7f)
	if b <= 0x7f {
		return Int
	}

	// FixMap (0x80 - 0x8f)
	if (b & 0xf0) == msgpack.Fixmap {
		return Map
	}

	// FixArray (0x90 - 0x9f)
	if (b & 0xf0) == msgpack.Fixarray {
		return Array
	}

	// FixStr (0xa0 - 0xbf)
	if (b & 0xe0) == msgpack.Fixstr {
		return Str
	}

	// Negative FixInt (0xe0 - 0xff)
	if b >= 0xe0 {
		return Int
	}

	// Now handle types with fixed identifiers
	switch b {
	case msgpack.Nil:
		return Nil
	case msgpack.False, msgpack.True:
		return Bool
	case msgpack.Bin8, msgpack.Bin16, msgpack.Bin32:
		return Bin
	case msgpack.Float32, msgpack.Float64:
		return Float
	case msgpack.Uint8, msgpack.Uint16, msgpack.Uint32, msgpack.Uint64:
		return Uint
	case msgpack.Int8, msgpack.Int16, msgpack.Int32, msgpack.Int64:
		return Int
	case msgpack.Str8, msgpack.Str16, msgpack.Str32:
		return Str
	case msgpack.Array16, msgpack.Array32:
		return Array
	case msgpack.Map16, msgpack.Map32:
		return Map
	case msgpack.Fixext4:
		return Time
	default:
		// Unknown type; you may want to handle this case explicitly
		return Nil
	}
}

func (v Value) Bool() (val bool) {
	val, _, _ = msgpack.ReadBool(v, 0)
	return
}

func (v Value) Int() (val int64) {
	val, _, _ = msgpack.ReadInt(v, 0)
	return
}

func (v Value) Uint() (val uint64) {
	val, _, _ = msgpack.ReadUint(v, 0)
	return
}

func (v Value) Float() float64 {
	if v[0]&0xf0 == msgpack.Float32 {
		val, _, _ := msgpack.ReadFloat32(v, 0)
		return float64(val)
	}

	val, _, _ := msgpack.ReadFloat64(v, 0)

	return val
}

func (v Value) Str() (val string) {
	val, _, _ = msgpack.ReadString(v, 0)
	return
}

func (v Value) StrCopy() (val string) {
	val, _, _ = msgpack.ReadStringCopy(v, 0)
	return
}

func (v Value) Bin() (val []byte) {
	val, _, _ = msgpack.ReadBinary(v, 0)
	return
}

// Get size of the value (excl. head) in bytes.
func (v Value) Size() int {
	l, _ := msgpack.GetMsgpackValueLength(v)
	return l - 1
}

// Returns number of items in array/map, or the number of bytes in all other cases.
func (v Value) Len() int {
	switch v.Type() {

	case Array:
		l, _, _ := msgpack.ReadArrayHeader(v, 0)
		return l

	case Map:
		l, _, _ := msgpack.ReadMapHeader(v, 0)
		return l
	}

	return v.Size()
}

func (v Value) Array() iter.Seq[Value] {
	return func(yield func(Value) bool) {
		len, off, _ := msgpack.ReadArrayHeader(v, 0)

		for i := 0; i < len; i++ {
			valLen, _ := msgpack.GetMsgpackValueLength(v[off:])

			if !yield(v[off : off+valLen]) {
				return
			}

			off += valLen
		}
	}
}

func (v Value) Map() iter.Seq2[Value, Value] {
	return func(yield func(Value, Value) bool) {
		len, off, _ := msgpack.ReadMapHeader(v, 0)

		for i := 0; i < len; i++ {
			// Key
			l, _ := msgpack.GetMsgpackValueLength(v[off:])
			key := v[off : off+l]
			off += l

			// Value
			l, _ = msgpack.GetMsgpackValueLength(v[off:])
			val := v[off : off+l]
			off += l

			if !yield(key, val) {
				return
			}
		}
	}
}

func (v Value) Time() (val time.Time) {
	val, _, err := msgpack.ReadTimestamp(v, 0)
	_ = err
	return
}

// Returns an allocated string representation of the value.
func (v Value) String() string {
	switch v.Type() {

	case Bool:
		return strconv.FormatBool(v.Bool())

	case Int:
		return strconv.FormatInt(v.Int(), 10)

	case Uint:
		return strconv.FormatUint(v.Uint(), 10)

	case Float:
		return strconv.FormatFloat(v.Float(), 'f', 6, 64)

	case Str:
		return v.StrCopy()

	case Bin:
		return "(" + strconv.Itoa(v.Size()) + " bytes)"

	case Array, Map:
		return "(" + strconv.Itoa(v.Len()) + " items)"

	case Time:
		return v.Time().Format(time.DateTime)
	}

	return "(nil)"
}
