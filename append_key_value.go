package fluentlog

import (
	"fmt"

	"github.com/webmafia/fluentlog/internal"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

func appendKeyValue(dst []byte, key string, value any) ([]byte, uint8) {
	switch val := value.(type) {

	case KeyValueAppender:
		return val.AppendKeyValue(dst, key)

	case internal.TextAppender:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendTextAppender(dst, val)

	case fmt.Stringer:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendString(dst, val.String())

	case []byte:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendBinary(dst, val)

	case bool:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendBool(dst, val)

	case float32:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendFloat(dst, float64(val))

	case float64:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendFloat(dst, val)

	case int:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendInt(dst, int64(val))

	case int8:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendInt(dst, int64(val))

	case int16:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendInt(dst, int64(val))

	case int32:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendInt(dst, int64(val))

	case int64:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendInt(dst, val)

	case uint:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendUint(dst, uint64(val))

	case uint8:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendUint(dst, uint64(val))

	case uint16:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendUint(dst, uint64(val))

	case uint32:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendUint(dst, uint64(val))

	case uint64:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendUint(dst, val)

	case string:
		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendString(dst, val)

	default:
		return dst, 0

	}

	return dst, 1
}

type KeyValueAppender interface {
	AppendKeyValue(dst []byte, key string) ([]byte, byte)
}
