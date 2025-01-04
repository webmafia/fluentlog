package msgpack

import (
	"math"
	"time"

	"github.com/webmafia/fluentlog/internal"
)

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

func AppendString(dst []byte, s string) []byte {
	l := len(s)
	switch {
	case l <= 31:
		dst = append(dst, 0xa0|byte(l))
	case l <= 0xFF:
		dst = append(dst, 0xd9, byte(l))
	case l <= 0xFFFF:
		dst = append(dst, 0xda, byte(l>>8), byte(l))
	default:
		dst = append(dst, 0xdb, byte(l>>24), byte(l>>16), byte(l>>8), byte(l))
	}
	return append(dst, s...)
}

func AppendTextAppender(dst []byte, s internal.TextAppender) []byte {
	return AppendUnknownString(dst, func(dst []byte) []byte {
		dst, _ = s.AppendText(dst)
		return dst
	})
}

func AppendUnknownString(dst []byte, fn func(dst []byte) []byte) []byte {

	// We don't know the length of the string, so assume the longest possible string.
	start := len(dst)
	dst = append(dst, 0xdb, 0, 0, 0, 0)
	sizeFrom := len(dst)
	dst = fn(dst)
	sizeTo := len(dst)
	l := sizeTo - sizeFrom

	// Now we know how many bytes that were appended - update the head accordingly.
	dst[start+1] = byte(l >> 24)
	dst[start+2] = byte(l >> 16)
	dst[start+3] = byte(l >> 8)
	dst[start+4] = byte(l)

	return dst
}

func AppendInt(dst []byte, i int64) []byte {
	if i >= 0 {
		return AppendUint(dst, uint64(i))
	}

	switch {
	case i >= -32:
		// Negative fixint
		return append(dst, 0xe0|byte(i+32))
	case i >= -128:
		// int8
		return append(dst, 0xd0, byte(i))
	case i >= -32768:
		// int16
		return append(dst, 0xd1, byte(i>>8), byte(i))
	case i >= -2147483648:
		// int32
		return append(dst, 0xd2, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		// int64
		return append(dst, 0xd3, byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32),
			byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	}
}

func AppendUint(dst []byte, i uint64) []byte {
	switch {
	case i <= 127:
		// Positive fixint
		return append(dst, byte(i))
	case i <= 255:
		// uint8
		return append(dst, 0xcc, byte(i))
	case i <= 65535:
		// uint16
		return append(dst, 0xcd, byte(i>>8), byte(i))
	case i <= 4294967295:
		// uint32
		return append(dst, 0xce, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		// uint64
		return append(dst, 0xcf, byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32),
			byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	}
}

func AppendNil(dst []byte) []byte {
	return append(dst, 0xc0)
}

func AppendBool(dst []byte, b bool) []byte {
	if b {
		return append(dst, 0xc3)
	}
	return append(dst, 0xc2)
}

func AppendBinary(dst []byte, data []byte) []byte {
	l := len(data)
	switch {
	case l <= 0xFF:
		dst = append(dst, 0xc4, byte(l))
	case l <= 0xFFFF:
		dst = append(dst, 0xc5, byte(l>>8), byte(l))
	default:
		dst = append(dst, 0xc6, byte(l>>24), byte(l>>16), byte(l>>8), byte(l))
	}
	return append(dst, data...)
}

func AppendBinaryAppender(dst []byte, s internal.BinaryAppender) []byte {
	return AppendUnknownBinary(dst, func(dst []byte) []byte {
		dst, _ = s.AppendBinary(dst)
		return dst
	})
}

func AppendUnknownBinary(dst []byte, fn func(dst []byte) []byte) []byte {

	// We don't know the length of the binary, so assume the longest possible binary.
	start := len(dst)
	dst = append(dst, 0xc6, 0, 0, 0, 0)
	sizeFrom := len(dst)
	dst = fn(dst)
	sizeTo := len(dst)
	l := sizeTo - sizeFrom

	// Now we know how many bytes that were appended - update the head accordingly.
	dst[start+1] = byte(l >> 24)
	dst[start+2] = byte(l >> 16)
	dst[start+3] = byte(l >> 8)
	dst[start+4] = byte(l)

	return dst
}

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

func AppendTimestamp(dst []byte, t time.Time) []byte {
	return AppendInt(dst, t.UTC().Unix())
}

func AppendExtendedTimestamp(dst []byte, t time.Time) []byte {
	s, ns := uint32(t.Unix()), uint32(t.Nanosecond())

	return append(dst,

		// Append the fixext8 header and type
		0xd7,
		0x00,

		// Append the seconds as a 32-bit big-endian integer
		byte(s>>24),
		byte(s>>16),
		byte(s>>8),
		byte(s),

		// Append the nanoseconds as a 32-bit big-endian integer
		byte(ns>>24),
		byte(ns>>16),
		byte(ns>>8),
		byte(ns),
	)
}
