package msgpack

import "github.com/webmafia/fluentlog/internal/msgpack/types"

func AppendArray(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		return append(dst, 0x90|byte(n))
	case n <= 0xFFFF:
		return append(dst, 0xdc, byte(n>>8), byte(n))
	default:
		return append(dst, 0xdd, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// ReadArrayHeader reads an array header from src starting at offset.
// It returns the length of the array and the new offset after reading.
func ReadArrayHeader(src []byte, offset int) (length int, newOffset int, err error) {
	typ, length, isValueLength := types.Get(src[offset])

	if typ != types.Array {
		err = expectedType(src[offset], types.Array)
		return
	}

	offset++

	if !isValueLength {
		l := length
		length = intFromBuf[int](src[offset : offset+l])
		offset += l
	}

	newOffset = offset
	return
}
