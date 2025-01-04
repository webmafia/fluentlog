package msgpack

import "github.com/webmafia/fluentlog/internal/msgpack/types"

func AppendMap(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		return append(dst, 0x80|byte(n))
	case n <= 0xFFFF:
		return append(dst, 0xde, byte(n>>8), byte(n))
	default:
		return append(dst, 0xdf, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// ReadMapHeader reads a map header from src starting at offset.
// It returns the number of key-value pairs and the new offset after reading.
func ReadMapHeader(src []byte, offset int) (length int, newOffset int, err error) {
	typ, length, isValueLength := types.Get(src[offset])

	if typ != types.Map {
		err = expectedType(src[offset], types.Map)
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
