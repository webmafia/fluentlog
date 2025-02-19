package msgpack

import "github.com/webmafia/fluentlog/pkg/msgpack/types"

// AppendMapHeader appends a map header with `n` key-value pairs to `dst` as a MessagePack-encoded value.
// Returns the updated byte slice.
func AppendMapHeader(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		return append(dst, 0x80|byte(n))
	case n <= 0xFFFF:
		return append(dst, 0xde, byte(n>>8), byte(n))
	default:
		return append(dst, 0xdf, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// ReadMapHeader reads a map header from `src` starting at `offset`.
// Returns the number of key-value pairs, the new offset, and an error if the header is invalid.
func ReadMapHeader(src []byte, offset int) (length int, newOffset int, err error) {
	if offset >= len(src) {
		err = ErrShortBuffer
		return
	}

	typ, length, isValueLength := types.Get(src[offset])

	if typ != types.Map {
		err = expectedType(src[offset], types.Map)
		return
	}

	offset++

	if !isValueLength {
		if offset+length > len(src) {
			return 0, offset, ErrShortBuffer
		}

		l := length
		length = intFromBuf[int](src[offset : offset+l])
		offset += l
	}

	newOffset = offset
	return
}
