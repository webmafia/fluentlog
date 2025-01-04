package msgpack

import (
	"encoding/binary"
)

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
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
