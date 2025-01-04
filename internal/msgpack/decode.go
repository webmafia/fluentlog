package msgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

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
