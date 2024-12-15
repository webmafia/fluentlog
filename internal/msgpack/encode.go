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
		dst = append(dst, 0xdc)
		return append(dst, byte(n>>8), byte(n))
	default:
		dst = append(dst, 0xdd)
		return append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

func AppendMap(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		return append(dst, 0x80|byte(n))
	case n <= 0xFFFF:
		dst = append(dst, 0xde)
		return append(dst, byte(n>>8), byte(n))
	default:
		dst = append(dst, 0xdf)
		return append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
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
		dst = append(dst, 0xd0)
		return append(dst, byte(i))
	case i >= -32768:
		// int16
		dst = append(dst, 0xd1)
		return append(dst, byte(i>>8), byte(i))
	case i >= -2147483648:
		// int32
		dst = append(dst, 0xd2)
		return append(dst, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		// int64
		dst = append(dst, 0xd3)
		return append(dst, byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32),
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
		dst = append(dst, 0xcc)
		return append(dst, byte(i))
	case i <= 65535:
		// uint16
		dst = append(dst, 0xcd)
		return append(dst, byte(i>>8), byte(i))
	case i <= 4294967295:
		// uint32
		dst = append(dst, 0xce)
		return append(dst, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		// uint64
		dst = append(dst, 0xcf)
		return append(dst, byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32),
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
	dst = append(dst, 0xca)
	return append(dst, byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

func AppendFloat64(dst []byte, f float64) []byte {
	bits := math.Float64bits(f)
	dst = append(dst, 0xcb)
	return append(dst,
		byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32),
		byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

func AppendTimestamp(dst []byte, t time.Time) []byte {
	return AppendInt(dst, t.UTC().Unix())
}

func AppendExtendedTimestamp(dst []byte, t time.Time) []byte {
	// Append the fixext8 header and type
	dst = append(dst, 0xd7, 0x00)

	// Append the seconds as a 32-bit big-endian integer
	seconds := uint32(t.Unix())
	dst = append(dst,
		byte(seconds>>24),
		byte(seconds>>16),
		byte(seconds>>8),
		byte(seconds),
	)

	// Append the nanoseconds as a 32-bit big-endian integer
	nanoseconds := uint32(t.Nanosecond())
	dst = append(dst,
		byte(nanoseconds>>24),
		byte(nanoseconds>>16),
		byte(nanoseconds>>8),
		byte(nanoseconds),
	)

	return dst
}
