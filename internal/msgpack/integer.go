// integer.go
package msgpack

import (
	"fmt"
	"math"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// ReadInt reads a MessagePack-encoded integer from `src` starting at `offset`.
// It handles both signed (TypeInt) and unsigned (TypeUint) integers.
// Returns the integer value as int64, the new offset, and an error if the data is invalid or incomplete.
func ReadInt(src []byte, offset int) (value int64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}

	c := src[offset]
	typ, length, isValueLength := types.Get(c)
	newOffset = offset + 1

	switch typ {
	case types.Int:
		if isValueLength && length == 0 {
			// Handle Negative FixInt (0xe0 to 0xff)
			if c >= 0xe0 {
				// Negative FixInt: single byte
				return intFromBuf[int64]([]byte{c}), newOffset, nil
			}
			// Unexpected: TypeInt with non-negative FixInt range
			return 0, offset, fmt.Errorf("unexpected TypeInt for non-negative fixint: 0x%02x", c)
		}
		// Handle multi-byte signed integers
		if newOffset+length > len(src) {
			return 0, offset, ErrShortBuffer
		}
		buf := src[newOffset : newOffset+length]
		value = intFromBuf[int64](buf)
		newOffset += length
		return value, newOffset, nil

	case types.Uint:
		if isValueLength && length == 0 {
			// Handle Positive FixInt (0x00 to 0x7f)
			if c <= 0x7f {
				// Positive FixInt: single byte
				return intFromBuf[int64](src[offset : offset+1]), newOffset, nil
			}
			// Unexpected: TypeUint with non-positive FixInt range
			return 0, offset, fmt.Errorf("unexpected TypeUint for non-positive fixint: 0x%02x", c)
		}
		// Handle multi-byte unsigned integers
		if newOffset+length > len(src) {
			return 0, offset, ErrShortBuffer
		}
		buf := src[newOffset : newOffset+length]
		uintVal := uintFromBuf[uint64](buf)
		newOffset += length
		if uintVal > math.MaxInt64 {
			return 0, offset, fmt.Errorf("uint64 value %d overflows int64", uintVal)
		}
		value = int64(uintVal)
		return value, newOffset, nil

	default:
		return 0, offset, fmt.Errorf("invalid int header byte: 0x%02x", c)
	}
}

// ReadUint reads a MessagePack-encoded unsigned integer from `src` starting at `offset`.
// It only accepts TypeUint and returns an error for any other type.
// Returns the unsigned integer value as uint64, the new offset, and an error if the data is invalid or incomplete.
func ReadUint(src []byte, offset int) (value uint64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}

	c := src[offset]
	typ, length, isValueLength := types.Get(c)
	newOffset = offset + 1

	if typ != types.Uint {
		return 0, offset, fmt.Errorf("invalid uint header byte: 0x%02x", c)
	}

	if isValueLength && length == 0 {
		// Handle Positive FixInt (0x00 to 0x7f)
		if c <= 0x7f {
			// Positive FixInt: single byte
			return uintFromBuf[uint64](src[offset : offset+1]), newOffset, nil
		}
		// Unexpected: TypeUint with non-positive FixInt range
		return 0, offset, fmt.Errorf("unexpected TypeUint for non-positive fixint: 0x%02x", c)
	}

	// Handle multi-byte unsigned integers
	if newOffset+length > len(src) {
		return 0, offset, ErrShortBuffer
	}

	buf := src[newOffset : newOffset+length]
	value = uintFromBuf[uint64](buf)
	newOffset += length
	return value, newOffset, nil
}

// AppendInt appends a MessagePack-encoded integer to the buffer.
// Positive integers are encoded as TypeUint, and negative integers as TypeInt.
// It uses the most compact representation based on the value.
func AppendInt(buf []byte, value int64) []byte {
	if value >= 0 {
		// Encode positive integers as TypeUint
		return AppendUint(buf, uint64(value))
	}

	// Encode negative integers as TypeInt
	switch {
	case value >= -32:
		// Negative FixInt: single byte
		return append(buf, byte(value))
	case value >= -128:
		// int8
		buf = append(buf, 0xd0)
		buf = append(buf, byte(value))
		return buf
	case value >= -32768:
		// int16
		buf = append(buf, 0xd1)
		// Manually encode without using make()
		b0 := byte(value >> 8)
		b1 := byte(value)
		buf = append(buf, b0, b1)
		return buf
	case value >= -2147483648:
		// int32
		buf = append(buf, 0xd2)
		// Manually encode without using make()
		b0 := byte(value >> 24)
		b1 := byte(value >> 16)
		b2 := byte(value >> 8)
		b3 := byte(value)
		buf = append(buf, b0, b1, b2, b3)
		return buf
	default:
		// int64
		buf = append(buf, 0xd3)
		// Manually encode without using make()
		b0 := byte(value >> 56)
		b1 := byte(value >> 48)
		b2 := byte(value >> 40)
		b3 := byte(value >> 32)
		b4 := byte(value >> 24)
		b5 := byte(value >> 16)
		b6 := byte(value >> 8)
		b7 := byte(value)
		buf = append(buf, b0, b1, b2, b3, b4, b5, b6, b7)
		return buf
	}
}

// AppendUint appends a MessagePack-encoded unsigned integer to the buffer.
// It uses the most compact representation based on the value.
func AppendUint(buf []byte, value uint64) []byte {
	switch {
	case value <= 127:
		// Positive FixInt: single byte
		return append(buf, byte(value))
	case value <= 255:
		// uint8
		buf = append(buf, 0xcc)
		return append(buf, byte(value))
	case value <= 65535:
		// uint16
		buf = append(buf, 0xcd)
		// Manually encode without using make()
		b0 := byte(value >> 8)
		b1 := byte(value)
		buf = append(buf, b0, b1)
		return buf
	case value <= 4294967295:
		// uint32
		buf = append(buf, 0xce)
		// Manually encode without using make()
		b0 := byte(value >> 24)
		b1 := byte(value >> 16)
		b2 := byte(value >> 8)
		b3 := byte(value)
		buf = append(buf, b0, b1, b2, b3)
		return buf
	default:
		// uint64
		buf = append(buf, 0xcf)
		// Manually encode without using make()
		b0 := byte(value >> 56)
		b1 := byte(value >> 48)
		b2 := byte(value >> 40)
		b3 := byte(value >> 32)
		b4 := byte(value >> 24)
		b5 := byte(value >> 16)
		b6 := byte(value >> 8)
		b7 := byte(value)
		buf = append(buf, b0, b1, b2, b3, b4, b5, b6, b7)
		return buf
	}
}

func readIntUnsafe[T Numeric](c byte, src []byte) (value T) {
	typ, _, _ := types.Get(c)

	switch typ {

	case types.Int:
		if c >= 0xe0 {
			return T(int8(c))
		}

		return T(intFromBuf[int64](src))

	case types.Uint:
		if c <= 0x7f {
			return T(int8(c))
		}

		return T(uintFromBuf[uint64](src))

	default:
		return 0
	}
}
