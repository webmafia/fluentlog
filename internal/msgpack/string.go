package msgpack

import (
	"strings"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

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

func ReadString(src []byte, offset int) (s string, newOffset int, err error) {
	typ, length, isValueLength := types.Get(src[offset])

	if typ != types.Str {
		err = expectedType(src[offset], types.Str)
		return
	}

	offset++

	if !isValueLength {
		l := length
		length = intFromBuf[int](src[offset : offset+l])
		offset += l
	}

	s = fast.BytesToString(src[offset : offset+length])
	newOffset = offset + length
	return
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

func AppendTextAppender(dst []byte, s internal.TextAppender) []byte {
	return AppendStringUnknownLength(dst, func(dst []byte) []byte {
		dst, _ = s.AppendText(dst)
		return dst
	})
}

func AppendStringUnknownLength(dst []byte, fn func(dst []byte) []byte) []byte {

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
