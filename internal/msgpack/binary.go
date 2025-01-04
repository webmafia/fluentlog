package msgpack

import (
	"fmt"

	"github.com/webmafia/fluentlog/internal"
)

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

func AppendBinaryAppender(dst []byte, s internal.BinaryAppender) []byte {
	return AppendBinaryUnknownLength(dst, func(dst []byte) []byte {
		dst, _ = s.AppendBinary(dst)
		return dst
	})
}

func AppendBinaryUnknownLength(dst []byte, fn func(dst []byte) []byte) []byte {

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
