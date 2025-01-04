package msgpack

import "fmt"

func AppendBool(dst []byte, b bool) []byte {
	if b {
		return append(dst, 0xc3)
	}
	return append(dst, 0xc2)
}

// ReadBool reads a boolean value from src starting at offset.
// It returns the boolean value and the new offset after reading.
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
		return false, offset, fmt.Errorf("invalid bool header byte: 0x%02x", b)
	}
	return value, offset, nil
}
