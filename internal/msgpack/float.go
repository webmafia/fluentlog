package msgpack

import (
	"math"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// AppendFloat appends a floating-point value (`f`) as a MessagePack-encoded float32 or float64 to `dst`.
// Returns the updated byte slice.
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

// ReadFloat reads a MessagePack-encoded floating-point value from `src` starting at `offset`.
// Returns the floating-point value, the new offset, and an error if the data is invalid or incomplete.
func ReadFloat(src []byte, offset int) (value float64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}

	typ, length, isValueLength := types.Get(src[offset])

	if typ != types.Float {
		err = expectedType(src[offset], types.Float)
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

	if offset+length > len(src) {
		return 0, offset, ErrShortBuffer
	}

	value = floatFromBuf[float64](src[offset : offset+length])
	offset += length

	return value, offset, nil
}
