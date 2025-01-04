package msgpack

import (
	"encoding/binary"
	"fmt"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

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

// ReadInt reads an integer value from src starting at offset.
// It returns the integer value and the new offset after reading.
func ReadInt(src []byte, offset int) (value int64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}

	c := src[offset]
	typ, length, isValueLength := types.Get(c)
	newOffset = offset + 1

	if typ != types.Int {
		return 0, offset, fmt.Errorf("invalid int header byte: 0x%02x", c)
	}

	if isValueLength && length == 0 {
		// Directly extract FixInt value
		if c >= 0xe0 { // Negative FixInt
			return int64(int8(c)), newOffset, nil
		}
		return int64(c), newOffset, nil // Positive FixInt
	}

	// Extract multi-byte integer
	if newOffset+length > len(src) {
		return 0, offset, ErrShortBuffer
	}

	switch length {
	case 1:
		value = int64(int8(src[newOffset]))
	case 2:
		value = int64(int16(binary.BigEndian.Uint16(src[newOffset:])))
	case 4:
		value = int64(int32(binary.BigEndian.Uint32(src[newOffset:])))
	case 8:
		value = int64(binary.BigEndian.Uint64(src[newOffset:]))
	default:
		return 0, offset, fmt.Errorf("unsupported int length: %d", length)
	}

	newOffset += length
	return
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

// ReadUint reads an unsigned integer value from src starting at offset.
// It returns the unsigned integer value and the new offset after reading.
func ReadUint(src []byte, offset int) (value uint64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}

	c := src[offset]
	typ, length, isValueLength := types.Get(c)
	newOffset = offset + 1

	if typ != types.Uint && typ != types.Int {
		return 0, offset, fmt.Errorf("invalid uint header byte: 0x%02x", c)
	}

	if isValueLength && length == 0 {
		// Directly extract FixInt value
		if c <= 0x7f { // Positive FixInt
			return uint64(c), newOffset, nil
		}
		return 0, offset, fmt.Errorf("negative value cannot be read as uint")
	}

	// Extract multi-byte unsigned integer
	if newOffset+length > len(src) {
		return 0, offset, ErrShortBuffer
	}

	switch length {
	case 1:
		value = uint64(src[newOffset])
	case 2:
		value = uint64(binary.BigEndian.Uint16(src[newOffset:]))
	case 4:
		value = uint64(binary.BigEndian.Uint32(src[newOffset:]))
	case 8:
		value = binary.BigEndian.Uint64(src[newOffset:])
	default:
		return 0, offset, fmt.Errorf("unsupported uint length: %d", length)
	}

	newOffset += length
	return
}
