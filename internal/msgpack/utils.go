package msgpack

import (
	"encoding/binary"
	"math"
)

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Float interface {
	~float32 | ~float64
}

type Numeric interface {
	Signed | Unsigned
}

// intFromBuf converts a byte slice to a signed integer value based on its length.
func intFromBuf[T Signed](buf []byte) T {
	switch len(buf) {
	case 1:
		return T(int8(buf[0]))
	case 2:
		return T(int16(binary.BigEndian.Uint16(buf)))
	case 4:
		return T(int32(binary.BigEndian.Uint32(buf)))
	case 8:
		return T(binary.BigEndian.Uint64(buf))
	default:
		return 0
	}
}

// uintFromBuf converts a byte slice to an unsigned integer value based on its length.
func uintFromBuf[T Unsigned](buf []byte) T {
	switch len(buf) {
	case 1:
		return T(buf[0])
	case 2:
		return T(binary.BigEndian.Uint16(buf))
	case 4:
		return T(binary.BigEndian.Uint32(buf))
	case 8:
		return T(binary.BigEndian.Uint64(buf))
	default:
		return 0
	}
}

// floatFromBuf converts a byte slice to a floating point value value based on its length.
func floatFromBuf[T Float](buf []byte) T {
	switch len(buf) {
	case 4:
		return T(math.Float32frombits(binary.BigEndian.Uint32(buf)))
	case 8:
		return T(math.Float64frombits(binary.BigEndian.Uint64(buf)))
	default:
		return 0
	}
}

// roundPow rounds to the next power of 2
// From: https://github.com/webmafia/fast
func roundPow(n int) int {
	if n <= 1 {
		return 1
	}

	// Start with the number minus one
	n--

	// Spread the highest set bit to the right
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16

	// Add one to get the next power of 2
	return n + 1
}
