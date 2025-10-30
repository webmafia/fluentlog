package msgpack

import (
	"encoding"
	"errors"
	"strconv"

	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

var _ encoding.TextAppender = (*Iterator)(nil)

// Appends a text representation of current value to b. Implements encoding.TextAppender.
func (iter *Iterator) AppendText(b []byte) ([]byte, error) {
	switch iter.typ {

	case types.Bool:
		b = strconv.AppendBool(b, iter.Bool())

	case types.Int:
		b = strconv.AppendInt(b, iter.Int(), 10)

	case types.Uint:
		b = strconv.AppendUint(b, iter.Uint(), 10)

	case types.Float:
		b = strconv.AppendFloat(b, iter.Float(), 'f', 6, 64)

	case types.Str:
		b = append(b, iter.Str()...)

	case types.Bin:
		b = append(b, iter.Bin()...)

	case types.Ext:
		b = iter.Time().AppendFormat(b, "2006-01-02 15:04:05 MST")

	case types.Array:
		b = iter.appendTextArray(b)

	case types.Map:
		b = iter.appendTextMap(b)

	default:
		return b, errors.New("invalid value type")
	}

	return b, nil
}

func (iter *Iterator) appendTextArray(b []byte) []byte {
	for i := range iter.Items() {
		if !iter.Next() {
			break
		}

		if i != 0 {
			b = append(b, '\n')
		}

		b, _ = iter.AppendText(b)
	}

	return b
}

func (iter *Iterator) appendTextMap(b []byte) []byte {
	for i := range iter.Items() {
		if !iter.Next() {
			break
		}

		if i != 0 {
			b = append(b, '\n')
		}

		b, _ = iter.AppendText(b)

		if !iter.Next() {
			break
		}

		b = append(b, ':', ' ')
		b, _ = iter.AppendText(b)
	}

	return b
}
