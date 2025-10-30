package msgpack

import (
	"encoding"
	"fmt"
	"math"
	"strings"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

// AppendString appends the string `s` as a MessagePack-encoded value to `dst`.
// Returns the updated byte slice.
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

// ReadString reads a MessagePack-encoded string from `src` starting at `offset`.
// Returns a zero-copy reference to the string from the `src` slice, the new offset,
// and an error if the data is invalid or incomplete.
func ReadString(src []byte, offset int) (s string, newOffset int, err error) {
	if offset >= len(src) {
		err = ErrShortBuffer
		return
	}

	typ, length, isValueLength := types.Get(src[offset])

	if typ == types.Bin {
		var v []byte
		v, newOffset, err = ReadBinary(src, offset)
		return fast.BytesToString(v), newOffset, err
	}

	if typ != types.Str {
		err = expectedType(src[offset], types.Str)
		return
	}

	offset++

	if !isValueLength {
		l := length
		if offset+l > len(src) {
			err = ErrShortBuffer
			return
		}

		// Read the length as an unsigned integer to prevent negative lengths
		uintLength := uintFromBuf[uint](src[offset : offset+l])
		if uintLength > math.MaxInt {
			err = fmt.Errorf("string length %d exceeds max int", uintLength)
			return
		}
		length = int(uintLength)
		offset += l
	}

	if offset+length > len(src) {
		err = ErrShortBuffer
		return
	}

	s = fast.BytesToString(src[offset : offset+length])
	newOffset = offset + length
	return
}

// ReadStringCopy reads a MessagePack-encoded string from `src` starting at `offset`.
// Returns a copy of the string, the new offset, and an error if the data is invalid or incomplete.
func ReadStringCopy(src []byte, offset int) (s string, newOffset int, err error) {
	s, newOffset, err = ReadString(src, offset)

	if err == nil {
		s = strings.Clone(s)
	}

	return
}

// AppendTextAppender appends a string to `dst` using a `TextAppender` and encodes it as a MessagePack string.
// Returns the updated byte slice.
func AppendTextAppender(dst []byte, s encoding.TextAppender) []byte {
	return AppendStringUnknownLength(dst, func(dst []byte) []byte {
		dst, _ = s.AppendText(dst)
		return dst
	})
}

// AppendStringUnknownLength appends a string with an unknown length to `dst` as a MessagePack-encoded value.
// The string data is appended using the provided function `fn`. Returns the updated byte slice.
func AppendStringUnknownLength(dst []byte, fn func(dst []byte) []byte) []byte {
	// We don't know the length of the string, so assume the longest possible string.
	start := len(dst)
	dst = append(dst, 0xdb, 0, 0, 0, 0)
	sizeFrom := len(dst)
	dst = fn(dst)
	sizeTo := len(dst)
	l := sizeTo - sizeFrom

	// Now we know how many bytes were appended - update the header accordingly.
	dst[start+1] = byte(l >> 24)
	dst[start+2] = byte(l >> 16)
	dst[start+3] = byte(l >> 8)
	dst[start+4] = byte(l)

	return dst
}

// AppendStringDynamic appends a MessagePack-encoded string to dst.
// The string data is produced by calling fn. The function uses the
// smallest header possible without allocating additional memory (aside from appending).
func AppendStringDynamic(dst []byte, fn func(dst []byte) []byte) []byte {
	// Reserve worst-case header: 5 bytes for str32.
	start := len(dst)
	dst = append(dst, 0xdb, 0, 0, 0, 0)

	// Append the string bytes using the provided function.
	sizeFrom := len(dst)
	dst = fn(dst)
	sizeTo := len(dst)
	l := sizeTo - sizeFrom // Actual string length

	// Determine the minimal header length needed.
	var headerLen int
	switch {
	case l <= 31:
		headerLen = 1 // fixstr: 1 byte header
	case l <= 255:
		headerLen = 2 // str8: marker + 1 byte length
	case l <= 65535:
		headerLen = 3 // str16: marker + 2 byte length
	default:
		headerLen = 5 // str32: marker + 4 byte length
	}

	// If a smaller header is possible, shift the string data left.
	shift := 5 - headerLen
	if shift > 0 {
		// Move the string bytes left by 'shift' bytes.
		copy(dst[start+headerLen:], dst[start+5:])
		// Adjust the slice length.
		dst = dst[:sizeTo-shift]
	}

	// Write the appropriate MessagePack header.
	switch headerLen {
	case 1:
		// fixstr: high 3 bits 101 and low 5 bits are length.
		dst[start] = 0xa0 | byte(l)
	case 2:
		dst[start] = 0xd9
		dst[start+1] = byte(l)
	case 3:
		dst[start] = 0xda
		dst[start+1] = byte(l >> 8)
		dst[start+2] = byte(l)
	case 5:
		dst[start] = 0xdb
		dst[start+1] = byte(l >> 24)
		dst[start+2] = byte(l >> 16)
		dst[start+3] = byte(l >> 8)
		dst[start+4] = byte(l)
	}

	return dst
}

// AppendStringUnknownLength appends a string with an unknown length, but max 255 characters.
// The string data is appended using the provided function `fn`. Returns the updated byte slice.
func AppendStringMax255(dst []byte, fn func(dst []byte) []byte) []byte {
	// We don't know the length of the string, so assume the longest possible string.
	start := len(dst)
	dst = append(dst, 0xd9, 0)
	sizeFrom := len(dst)
	dst = fn(dst)
	sizeTo := len(dst)
	l := sizeTo - sizeFrom

	if l > 255 {
		l = 255
		dst = dst[:sizeFrom+l]
	}

	// Now we know how many bytes were appended - update the header accordingly.
	dst[start+1] = byte(l)

	return dst
}
