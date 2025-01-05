package msgpack

import "github.com/webmafia/fluentlog/internal/msgpack/types"

// AppendArrayHeader appends a MessagePack array header to `dst` based on the number of elements `n`.
// Returns the updated byte slice.
func AppendArrayHeader(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		return append(dst, 0x90|byte(n))
	case n <= 0xFFFF:
		return append(dst, 0xdc, byte(n>>8), byte(n))
	default:
		return append(dst, 0xdd, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// ReadArrayHeader reads a MessagePack array header from `src` starting at `offset`.
// Returns the array length, the new offset, and an error if the header is invalid.
func ReadArrayHeader(src []byte, offset int) (length int, newOffset int, err error) {
	if offset >= len(src) {
		err = ErrShortBuffer
		return
	}

	typ, length, isValueLength := types.Get(src[offset])

	if typ != types.Array {
		err = expectedType(src[offset], types.Array)
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
