package fluentlog

import "github.com/webmafia/fast/buffer"

func appendArgs(b *buffer.Buffer, args []any) (n uint8) {
	var key string

	for i := range args {
		if key == "" {
			if k, ok := args[i].(string); ok {
				key = k
				continue
			}
		}

		var nn uint8
		b.B, nn = appendKeyValue(b.B, key, args[i])
		key = ""
		n += nn
	}

	return
}
