package msgpack

import (
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// AppendBool appends a boolean value (`true` or `false`) as a MessagePack-encoded boolean to `dst`.
// Returns the updated byte slice.
func AppendBool(dst []byte, b bool) []byte {
	if b {
		return append(dst, 0xc3)
	}
	return append(dst, 0xc2)
}

// ReadBool reads a MessagePack-encoded boolean value from `src` starting at `offset`.
// Returns the boolean value, the new offset, and an error if the header is invalid or the buffer is too short.
func ReadBool(src []byte, offset int) (value bool, newOffset int, err error) {
	if offset >= len(src) {
		return false, offset, ErrShortBuffer
	}
	b := src[offset]
	offset++
	if b == 0xc2 {
		value = false
	} else if b == 0xc3 {
		value = true
	} else {
		return false, offset, expectedType(b, types.Bool)
	}
	return value, offset, nil
}

func readBoolUnsafe(c byte) bool {
	return c == 0xc3
}
