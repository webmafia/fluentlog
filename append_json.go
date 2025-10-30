package fluentlog

import (
	"github.com/segmentio/encoding/json"
	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

func appendJSON(dst []byte, v any) []byte {
	return msgpack.AppendBinaryUnknownLength(dst, func(dst []byte) []byte {
		newDst, err := json.Append(dst, fast.Noescape(v), 0)

		if err != nil {
			return dst
		}

		return newDst
	})
}
