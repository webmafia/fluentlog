package msgpack

import (
	"encoding/hex"
	"fmt"
	"iter"
	"strconv"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

var (
	_ fmt.Stringer        = (Value)(nil)
	_ fast.TextAppender   = (Value)(nil)
	_ fast.BinaryAppender = (Value)(nil)
	_ fast.JsonAppender   = (Value)(nil)
)

type Value []byte

func (v Value) IsZero() bool {
	return len(v) == 0
}

func (v Value) Type() (t types.Type) {
	if len(v) == 0 {
		return types.Nil
	}

	t, _, _ = types.Get(v[0])
	return
}

func (v Value) Array() iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if len(v) == 0 {
			return
		}

		var offset int
		typ, length, isValueLength := types.Get(v[offset])

		if typ != types.Array {
			return
		}

		offset++

		if !isValueLength {
			l := length
			length = intFromBuf[int](v[offset : offset+l])
			offset += l
		}

		v = v[offset:]

		for range length {
			next := v.BytesLen()

			if !yield(v[:next]) {
				return
			}

			v = v[next:]
		}
	}
}

func (v Value) Map() iter.Seq2[Value, Value] {
	return func(yield func(key, val Value) bool) {
		if len(v) == 0 {
			return
		}

		var offset int
		typ, length, isValueLength := types.Get(v[offset])

		if typ != types.Map {
			return
		}

		offset++

		if !isValueLength {
			l := length
			length = intFromBuf[int](v[offset : offset+l])
			offset += l
		}

		v = v[offset:]

		for range length {
			next := v.BytesLen()
			key := v[:next]
			v = v[next:]

			next = v.BytesLen()
			val := v[:next]
			v = v[next:]

			if !yield(key, val) {
				return
			}
		}
	}
}

// Returns the total number of bytes for the value. Head + body is included
// for all types except array and maps, where the body is excluded.
func (v Value) BytesLen() (l int) {
	if len(v) == 0 {
		return
	}

	var offset int
	typ, length, isValueLength := types.Get(v[offset])

	offset++

	if !isValueLength {
		l := length
		length = intFromBuf[int](v[offset : offset+l])
		offset += l
	}

	if typ != types.Array && typ != types.Map {
		offset += length
	}

	return offset
}

// Whether the value has its full bytes.
// func (v Value) IsComplete() bool {
// 	l := len(v)
// 	bl := v.BytesLen()
// 	return l >= bl
// }

func (v Value) Len() (l int) {
	if len(v) == 0 {
		return
	}

	_, l, isValueLength := types.Get(v[0])

	if !isValueLength {
		l = intFromBuf[int](v[1 : 1+l])
	}

	return
}

func (v Value) Bool() (val bool) {
	val, _, _ = ReadBool(v, 0)
	return
}

func (v Value) Int() (val int64) {
	val, _, _ = ReadInt(v, 0)
	return
}

func (v Value) Uint() (val uint64) {
	val, _, _ = ReadUint(v, 0)
	return
}

func (v Value) Float() (val float64) {
	val, _, _ = ReadFloat(v, 0)
	return
}

func (v Value) Str() (val string) {
	val, _, _ = ReadString(v, 0)
	return
}

func (v Value) StrCopy() (val string) {
	val, _, _ = ReadStringCopy(v, 0)
	return
}

func (v Value) Bin() (val []byte) {
	val, _, _ = ReadBinary(v, 0)
	return
}

func (v Value) Timestamp() (val time.Time) {
	val, _, _ = ReadTimestamp(v, 0)
	return
}

// Returns an allocated string representation of the value.
func (v Value) String() string {
	switch v.Type() {

	case types.Bool:
		return strconv.FormatBool(v.Bool())

	case types.Int:
		return strconv.FormatInt(v.Int(), 10)

	case types.Uint:
		return strconv.FormatUint(v.Uint(), 10)

	case types.Float:
		return strconv.FormatFloat(v.Float(), 'f', 6, 64)

	case types.Str:
		return v.StrCopy()

	case types.Bin:
		return hex.EncodeToString(v.Bin())

	case types.Array:
		return "Array<" + strconv.Itoa(v.Len()) + ">"

	case types.Map:
		return "Map<" + strconv.Itoa(v.Len()) + ">"

	case types.Ext:
		return v.Timestamp().Format(time.DateTime)
	}

	return ""
}

// AppendText implements fast.TextAppender.
func (v Value) AppendText(b []byte) ([]byte, error) {
	switch v.Type() {

	case types.Bool:
		return strconv.AppendBool(b, v.Bool()), nil

	case types.Int:
		return strconv.AppendInt(b, v.Int(), 10), nil

	case types.Uint:
		return strconv.AppendUint(b, v.Uint(), 10), nil

	case types.Float:
		return strconv.AppendFloat(b, v.Float(), 'f', 6, 64), nil

	case types.Str:
		return append(b, v.Str()...), nil

	case types.Bin:
		return hex.AppendEncode(b, v.Bin()), nil

	case types.Array:
		b = append(b, "Array<"...)
		b = strconv.AppendInt(b, int64(v.Len()), 10)
		return append(b, '>'), nil

	case types.Map:
		b = append(b, "Map<"...)
		b = strconv.AppendInt(b, int64(v.Len()), 10)
		return append(b, '>'), nil

	case types.Ext:
		return v.Timestamp().AppendFormat(b, time.DateTime), nil
	}

	return b, ErrInvalidFormat
}

// AppendBinary implements fast.BinaryAppender.
func (v Value) AppendBinary(b []byte) ([]byte, error) {
	return append(b, v...), nil
}

// AppendJson implements fast.JsonAppender.
func (v Value) AppendJson(b []byte) ([]byte, error) {
	panic("not implemented")
}
