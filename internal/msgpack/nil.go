package msgpack

import "fmt"

func AppendNil(dst []byte) []byte {
	return append(dst, 0xc0)
}

// ReadNil reads a nil value from src starting at offset.
// It returns the new offset after reading.
func ReadNil(src []byte, offset int) (newOffset int, err error) {
	if offset >= len(src) {
		return offset, ErrShortBuffer
	}
	if src[offset] != 0xc0 {
		return offset, fmt.Errorf("expected nil (0xc0), got 0x%02x", src[offset])
	}
	return offset + 1, nil
}
