package msgpack

import (
	"math"
	"time"
)

// AppendArray appends an array header to dst, choosing the smallest possible format.
// It supports fixarray, array16, and array32.
func AppendArray(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		// fixarray
		return append(dst, 0b10010000|byte(n))
	case n <= 0xFFFF:
		// array 16
		dst = append(dst, 0xdc)
		return append(dst, byte(n>>8), byte(n))
	default:
		// array 32
		dst = append(dst, 0xdd)
		return append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// AppendMap appends a map header to dst, choosing the smallest possible format.
// It supports fixmap, map16, and map32.
func AppendMap(dst []byte, n int) []byte {
	switch {
	case n <= 15:
		// fixmap
		return append(dst, 0b10000000|byte(n))
	case n <= 0xFFFF:
		// map 16
		dst = append(dst, 0xde)
		return append(dst, byte(n>>8), byte(n))
	default:
		// map 32
		dst = append(dst, 0xdf)
		return append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// AppendString appends a string to dst, choosing the smallest possible format.
// It supports fixstr, str8, str16, and str32.
func AppendString(dst []byte, s string) []byte {
	l := len(s)
	switch {
	case l <= 31:
		// fixstr: 101XXXXX
		dst = append(dst, 0b10100000|byte(l&0b00011111))
	case l <= 0xFF:
		// str8
		dst = append(dst, 0xd9, byte(l))
	case l <= 0xFFFF:
		// str16
		dst = append(dst, 0xda, byte(l>>8), byte(l))
	default:
		// str32
		dst = append(dst, 0xdb, byte(l>>24), byte(l>>16), byte(l>>8), byte(l))
	}
	return append(dst, s...)
}

// AppendInt appends an integer to dst, choosing the smallest possible format.
func AppendInt(dst []byte, i int64) []byte {
	switch {
	case i >= 0 && i <= 127:
		// positive fixint
		return append(dst, byte(i))
	case i >= -32 && i <= -1:
		// negative fixint
		return append(dst, 0xe0|byte(i+32))
	case i >= -128 && i <= 127:
		// int 8
		dst = append(dst, 0xd0, byte(int8(i)))
	case i >= -32768 && i <= 32767:
		// int 16
		dst = append(dst, 0xd1, byte(i>>8), byte(i))
	case i >= -2147483648 && i <= 2147483647:
		// int 32
		dst = append(dst, 0xd2, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		// int 64
		dst = append(dst, 0xd3,
			byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32),
			byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	}
	return dst
}

// AppendUint appends an unsigned integer to dst, choosing the smallest possible format.
func AppendUint(dst []byte, u uint64) []byte {
	switch {
	case u <= 127:
		// positive fixint
		return append(dst, byte(u))
	case u <= 0xFF:
		// uint 8
		dst = append(dst, 0xcc, byte(u))
	case u <= 0xFFFF:
		// uint 16
		dst = append(dst, 0xcd, byte(u>>8), byte(u))
	case u <= 0xFFFFFFFF:
		// uint 32
		dst = append(dst, 0xce, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
	default:
		// uint 64
		dst = append(dst, 0xcf,
			byte(u>>56), byte(u>>48), byte(u>>40), byte(u>>32),
			byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
	}
	return dst
}

// AppendNil appends a nil value to dst.
func AppendNil(dst []byte) []byte {
	return append(dst, 0xc0)
}

// AppendBool appends a boolean value to dst.
func AppendBool(dst []byte, b bool) []byte {
	if b {
		return append(dst, 0xc3) // true
	}
	return append(dst, 0xc2) // false
}

// AppendEventTime appends an EventTime extension object to dst.
// EventTime format uses fixext8 with type 0x00.
func AppendEventTime(dst []byte, t time.Time) []byte {
	// EventTime is represented by 8 bytes:
	// 4 bytes: seconds since epoch (big-endian)
	// 4 bytes: nanoseconds (big-endian)
	sec := uint32(t.Unix())
	nsec := uint32(t.Nanosecond())
	dst = append(dst, 0xd7, 0x00) // fixext8 and type 0x00
	dst = append(dst,
		byte(sec>>24), byte(sec>>16), byte(sec>>8), byte(sec),
		byte(nsec>>24), byte(nsec>>16), byte(nsec>>8), byte(nsec))
	return dst
}

// AppendExt appends an ext object to dst, choosing the smallest possible format.
// For data lengths 1, 2, 4, 8, or 16, it uses fixext formats.
func AppendExt(dst []byte, typ int8, data []byte) []byte {
	l := len(data)
	switch l {
	case 1:
		dst = append(dst, 0xd4, byte(typ))
	case 2:
		dst = append(dst, 0xd5, byte(typ))
	case 4:
		dst = append(dst, 0xd6, byte(typ))
	case 8:
		dst = append(dst, 0xd7, byte(typ))
	case 16:
		dst = append(dst, 0xd8, byte(typ))
	default:
		switch {
		case l <= 0xFF:
			// ext 8
			dst = append(dst, 0xc7, byte(l), byte(typ))
		case l <= 0xFFFF:
			// ext 16
			dst = append(dst, 0xc8, byte(l>>8), byte(l), byte(typ))
		default:
			// ext 32
			dst = append(dst, 0xc9,
				byte(l>>24), byte(l>>16), byte(l>>8), byte(l), byte(typ))
		}
	}
	return append(dst, data...)
}

// AppendBinary appends binary data to dst, choosing the smallest possible format.
// It supports bin8, bin16, and bin32.
func AppendBinary(dst []byte, data []byte) []byte {
	l := len(data)
	switch {
	case l <= 0xFF:
		// bin 8
		dst = append(dst, 0xc4, byte(l))
	case l <= 0xFFFF:
		// bin 16
		dst = append(dst, 0xc5, byte(l>>8), byte(l))
	default:
		// bin 32
		dst = append(dst, 0xc6,
			byte(l>>24), byte(l>>16), byte(l>>8), byte(l))
	}
	return append(dst, data...)
}

// AppendFloat32 appends a 32-bit floating point number to dst.
func AppendFloat32(dst []byte, f float32) []byte {
	bits := math.Float32bits(f)
	dst = append(dst, 0xca) // float 32
	return append(dst,
		byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

// AppendFloat64 appends a 64-bit floating point number to dst.
func AppendFloat64(dst []byte, f float64) []byte {
	bits := math.Float64bits(f)
	dst = append(dst, 0xcb) // float 64
	return append(dst,
		byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32),
		byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}
