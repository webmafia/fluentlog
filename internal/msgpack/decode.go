package msgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

var (
	// ErrShortBuffer is returned when the byte slice is too short to read the expected data.
	ErrShortBuffer = io.ErrShortBuffer
	// ErrInvalidFormat is returned when the data does not conform to the expected MessagePack format.
	ErrInvalidFormat = errors.New("invalid MessagePack format")
)

func expectType(c byte, expected types.Type) (err error) {
	if types.Get(c) != expected {
		err = fmt.Errorf("invalid %s header byte: 0x%02x", expected, c)
	}

	return
}

// ReadArrayHeader reads an array header from src starting at offset.
// It returns the length of the array and the new offset after reading.
func ReadArrayHeader(src []byte, offset int) (length int, newOffset int, err error) {
	if err = expectType(src[offset], types.Array); err != nil {
		return
	}

	return readLen(src, offset)
}

// ReadMapHeader reads a map header from src starting at offset.
// It returns the number of key-value pairs and the new offset after reading.
func ReadMapHeader(src []byte, offset int) (length int, newOffset int, err error) {
	if err = expectType(src[offset], types.Map); err != nil {
		return
	}

	return readLen(src, offset)
}

func ReadString(src []byte, offset int) (s string, newOffset int, err error) {
	if err = expectType(src[offset], types.Str); err != nil {
		return
	}

	length, offset, err := readLen(src, offset)

	if err != nil {
		return
	}

	// Extract the string and return
	s = fast.BytesToString(src[offset : offset+int(length)])
	newOffset = offset + int(length)
	return
}

func readLen(src []byte, offset int) (length int, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}

	// Get the length of the string or the length of the "length field"
	length, isValLen := types.GetLength(src[offset])

	// Advance to the next byte after the type
	offset++

	// If the length is encoded in additional bytes, decode it
	if !isValLen {
		// Ensure enough bytes are available for the length field
		if len(src) < offset+int(length) {
			err = ErrShortBuffer
			return
		}

		// Decode the length from the next `length` bytes
		decodedLength := types.GetInt(src[offset : offset+int(length)])
		offset += int(length) // Advance past the length field
		length = decodedLength
	}

	// Ensure enough bytes are available for the string data
	if len(src) < offset+int(length) {
		err = ErrShortBuffer
		return
	}

	return length, offset, nil
}

// ReadString reads a string from src starting at offset.
// It returns the string and the new offset after reading.
func ReadStringCopy(src []byte, offset int) (s string, newOffset int, err error) {
	s, newOffset, err = ReadString(src, offset)

	if err == nil {
		s = strings.Clone(s)
	}

	return
}

// ReadInt reads an integer value from src starting at offset.
// It returns the integer value and the new offset after reading.
func ReadInt(src []byte, offset int) (value int64, newOffset int, err error) {
	if typ := types.Get(src[offset]); typ != types.Int && typ != types.Uint {
		err = fmt.Errorf("invalid integer header byte: 0x%02x", src[offset])
		return
	}

	length, offset, err := readLen(src, offset)

	if err == nil {
		value = int64(types.GetInt(src[offset : offset+length]))
		offset += length
	}

	return value, offset, nil
}

// ReadUint reads an unsigned integer value from src starting at offset.
// It returns the unsigned integer value and the new offset after reading.
func ReadUint(src []byte, offset int) (value uint64, newOffset int, err error) {
	if offset >= len(src) {
		return 0, offset, ErrShortBuffer
	}
	b := src[offset]
	offset++
	switch {
	case b <= 0x7f:
		// positive fixint
		value = uint64(b)
	case b == 0xcc:
		// uint8
		if offset >= len(src) {
			return 0, offset, ErrShortBuffer
		}
		value = uint64(src[offset])
		offset++
	case b == 0xcd:
		// uint16
		if offset+1 >= len(src) {
			return 0, offset, ErrShortBuffer
		}
		value = uint64(src[offset])<<8 | uint64(src[offset+1])
		offset += 2
	case b == 0xce:
		// uint32
		if offset+3 >= len(src) {
			return 0, offset, ErrShortBuffer
		}
		value = uint64(src[offset])<<24 | uint64(src[offset+1])<<16 | uint64(src[offset+2])<<8 | uint64(src[offset+3])
		offset += 4
	case b == 0xcf:
		// uint64
		if offset+7 >= len(src) {
			return 0, offset, ErrShortBuffer
		}
		value = uint64(src[offset])<<56 | uint64(src[offset+1])<<48 | uint64(src[offset+2])<<40 | uint64(src[offset+3])<<32 |
			uint64(src[offset+4])<<24 | uint64(src[offset+5])<<16 | uint64(src[offset+6])<<8 | uint64(src[offset+7])
		offset += 8
	default:
		return 0, offset, fmt.Errorf("invalid uint header byte: 0x%02x", b)
	}
	return value, offset, nil
}

// ReadNil reads a nil value from src starting at offset.
// It returns the new offset after reading.
func ReadNil(src []byte, offset int) (newOffset int, err error) {
	if offset >= len(src) {
		return offset, ErrShortBuffer
	}
	if src[offset] != 0xc0 {
		return offset, fmt.Errorf("expected nil (0xc0), got 0x%02x", src[offset])
	}
	return offset + 1, nil
}

// ReadBool reads a boolean value from src starting at offset.
// It returns the boolean value and the new offset after reading.
func ReadBool(src []byte, offset int) (value bool, newOffset int, err error) {
	if offset >= len(src) {
		return false, offset, ErrShortBuffer
	}
	b := src[offset]
	offset++
	if b == 0xc2 {
		value = false
	} else if b == 0xc3 {
		value = true
	} else {
		return false, offset, fmt.Errorf("invalid bool header byte: 0x%02x", b)
	}
	return value, offset, nil
}

// ReadTimestamp reads a timestamp value from the given byte slice starting at the specified offset.
// It supports both EventTime (ext type 0) and integer timestamps.
func ReadTimestamp(src []byte, offset int) (t time.Time, newOffset int, err error) {
	if offset >= len(src) {
		err = io.ErrUnexpectedEOF
		return
	}

	b := src[offset]
	var s, ns int64

	switch b {
	case 0xd7: // fixext8
		if offset+10 > len(src) {
			err = io.ErrUnexpectedEOF
			return
		}

		if src[offset+1] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(src[offset+2 : offset+6])))
		ns = int64(int32(binary.BigEndian.Uint32(src[offset+6 : offset+10])))
		newOffset = offset + 10

	case 0xc7: // ext8
		if offset+11 > len(src) {
			err = io.ErrUnexpectedEOF
			return
		}

		if src[offset+1] != 0x08 || src[offset+2] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(src[offset+3 : offset+7])))
		ns = int64(int32(binary.BigEndian.Uint32(src[offset+7 : offset+11])))
		newOffset = offset + 11

	default:
		var intVal int64
		intVal, newOffset, err = ReadInt(src, offset)
		if err != nil {
			return
		}
		s = intVal
	}

	t = time.Unix(s, ns)
	return
}

// ReadExt reads an extension object from src starting at offset.
// It returns the type, data, and the new offset after reading.
func ReadExt(src []byte, offset int) (typ int8, data []byte, newOffset int, err error) {
	if offset >= len(src) {
		return 0, nil, offset, ErrShortBuffer
	}
	b := src[offset]
	offset++
	var length int
	switch b {
	case 0xd4:
		// fixext 1
		length = 1
	case 0xd5:
		// fixext 2
		length = 2
	case 0xd6:
		// fixext 4
		length = 4
	case 0xd7:
		// fixext 8
		length = 8
	case 0xd8:
		// fixext 16
		length = 16
	case 0xc7:
		// ext 8
		if offset >= len(src) {
			return 0, nil, offset, ErrShortBuffer
		}
		length = int(src[offset])
		offset++
	case 0xc8:
		// ext 16
		if offset+1 >= len(src) {
			return 0, nil, offset, ErrShortBuffer
		}
		length = int(src[offset])<<8 | int(src[offset+1])
		offset += 2
	case 0xc9:
		// ext 32
		if offset+3 >= len(src) {
			return 0, nil, offset, ErrShortBuffer
		}
		length = int(src[offset])<<24 | int(src[offset+1])<<16 | int(src[offset+2])<<8 | int(src[offset+3])
		offset += 4
	default:
		return 0, nil, offset, fmt.Errorf("invalid ext header byte: 0x%02x", b)
	}
	if offset >= len(src) {
		return 0, nil, offset, ErrShortBuffer
	}
	typ = int8(src[offset])
	offset++
	if offset+length > len(src) {
		return 0, nil, offset, ErrShortBuffer
	}
	data = src[offset : offset+length]
	offset += length
	return typ, data, offset, nil
}

// ReadBinary reads binary data from src starting at offset.
// It returns the data and the new offset after reading.
func ReadBinary(src []byte, offset int) (data []byte, newOffset int, err error) {
	if offset >= len(src) {
		return nil, offset, ErrShortBuffer
	}
	b := src[offset]
	offset++
	var length int
	switch b {
	case 0xc4:
		// bin 8
		if offset >= len(src) {
			return nil, offset, ErrShortBuffer
		}
		length = int(src[offset])
		offset++
	case 0xc5:
		// bin 16
		if offset+1 >= len(src) {
			return nil, offset, ErrShortBuffer
		}
		length = int(src[offset])<<8 | int(src[offset+1])
		offset += 2
	case 0xc6:
		// bin 32
		if offset+3 >= len(src) {
			return nil, offset, ErrShortBuffer
		}
		length = int(src[offset])<<24 | int(src[offset+1])<<16 | int(src[offset+2])<<8 | int(src[offset+3])
		offset += 4
	default:
		return nil, offset, fmt.Errorf("invalid binary header byte: 0x%02x", b)
	}
	if offset+length > len(src) {
		return nil, offset, ErrShortBuffer
	}
	data = src[offset : offset+length]
	offset += length
	return data, offset, nil
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

func Skip(src []byte, offset int) (newOffset int, err error) {
	typ := types.Get(src[offset])
	length, offset, err := readLen(src, offset)

	if err != nil {
		return
	}

	if typ == types.Map {
		length *= 2
	} else if typ != types.Array {
		return offset + length, nil
	}

	for range length {
		if offset, err = Skip(src, offset); err != nil {
			return
		}
	}

	return offset, nil
}
