package msgpack

import (
	"fmt"
	"math"
)

func AppendFloat32(dst []byte, f float32) []byte {
	bits := math.Float32bits(f)
	return append(dst, 0xca, byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

func AppendFloat64(dst []byte, f float64) []byte {
	bits := math.Float64bits(f)
	return append(dst, 0xcb,
		byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32),
		byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

// ReadFloat32 reads a 32-bit floating point number from src starting at offset.
// It returns the float32 value and the new offset after reading.
func ReadFloat32(src []byte, offset int) (value float32, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}
	if src[offset] != 0xca {
		return 0, offset, fmt.Errorf("expected float32 (0xca), got 0x%02x", src[offset])
	}
	offset++
	if offset+3 >= len(src) {
		return 0, offset, ErrShortBuffer
	}
	bits := uint32(src[offset])<<24 | uint32(src[offset+1])<<16 | uint32(src[offset+2])<<8 | uint32(src[offset+3])
	value = math.Float32frombits(bits)
	offset += 4
	return value, offset, nil
}

// ReadFloat64 reads a 64-bit floating point number from src starting at offset.
// It returns the float64 value and the new offset after reading.
func ReadFloat64(src []byte, offset int) (value float64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}
	if src[offset] != 0xcb {
		return 0, offset, fmt.Errorf("expected float64 (0xcb), got 0x%02x", src[offset])
	}
	offset++
	if offset+7 >= len(src) {
		return 0, offset, ErrShortBuffer
	}
	bits := uint64(src[offset])<<56 | uint64(src[offset+1])<<48 | uint64(src[offset+2])<<40 | uint64(src[offset+3])<<32 |
		uint64(src[offset+4])<<24 | uint64(src[offset+5])<<16 | uint64(src[offset+6])<<8 | uint64(src[offset+7])
	value = math.Float64frombits(bits)
	offset += 8
	return value, offset, nil
}
