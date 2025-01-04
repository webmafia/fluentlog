package msgpack

import (
	"fmt"
	"math"
)

func AppendFloat(dst []byte, f float64) []byte {
	// Check if the float can be represented as float32 without loss
	if f32 := float32(f); float64(f32) == f {
		// Encode as float32
		bits := math.Float32bits(f32)
		return append(dst, 0xca, byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
	}
	// Encode as float64
	bits := math.Float64bits(f)
	return append(dst, 0xcb,
		byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32),
		byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

func ReadFloat(src []byte, offset int) (value float64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}
	switch src[offset] {
	case 0xca: // float32
		offset++
		if offset+3 >= len(src) {
			return 0, offset, ErrShortBuffer
		}
		bits := uint32(src[offset])<<24 | uint32(src[offset+1])<<16 | uint32(src[offset+2])<<8 | uint32(src[offset+3])
		value = float64(math.Float32frombits(bits))
		offset += 4
	case 0xcb: // float64
		offset++
		if offset+7 >= len(src) {
			return 0, offset, ErrShortBuffer
		}
		bits := uint64(src[offset])<<56 | uint64(src[offset+1])<<48 | uint64(src[offset+2])<<40 | uint64(src[offset+3])<<32 |
			uint64(src[offset+4])<<24 | uint64(src[offset+5])<<16 | uint64(src[offset+6])<<8 | uint64(src[offset+7])
		value = math.Float64frombits(bits)
		offset += 8
	default:
		return 0, offset, fmt.Errorf("expected float32 (0xca) or float64 (0xcb), got 0x%02x", src[offset])
	}
	return value, offset, nil
}
