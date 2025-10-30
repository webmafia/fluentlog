package msgpack

import (
	"encoding"
	"fmt"
)

func AppendAny(dst []byte, v any) []byte {
	switch val := v.(type) {

	case []byte:
		dst = AppendBinary(dst, val)

	case bool:
		dst = AppendBool(dst, val)

	case float32:
		dst = AppendFloat(dst, float64(val))

	case float64:
		dst = AppendFloat(dst, val)

	case int:
		dst = AppendInt(dst, int64(val))

	case int8:
		dst = AppendInt(dst, int64(val))

	case int16:
		dst = AppendInt(dst, int64(val))

	case int32:
		dst = AppendInt(dst, int64(val))

	case int64:
		dst = AppendInt(dst, val)

	case uint:
		dst = AppendUint(dst, uint64(val))

	case uint8:
		dst = AppendUint(dst, uint64(val))

	case uint16:
		dst = AppendUint(dst, uint64(val))

	case uint32:
		dst = AppendUint(dst, uint64(val))

	case uint64:
		dst = AppendUint(dst, val)

	case string:
		dst = AppendString(dst, val)

	case encoding.TextAppender:
		dst = AppendTextAppender(dst, val)

	case fmt.Stringer:
		dst = AppendString(dst, val.String())

	}

	return dst
}
