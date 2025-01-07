package msgpack

import (
	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// AppendBinary appends a MessagePack binary header and the binary `data` to `dst`.
// Returns the updated byte slice.
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

// ReadBinary reads a MessagePack binary object from `src` starting at `offset`.
// Returns the binary data, the new offset, and an error if the header is invalid or the buffer is too short.
func ReadBinary(src []byte, offset int) (data []byte, newOffset int, err error) {
	if offset >= len(src) {
		return nil, offset, ErrShortBuffer
	}

	typ, length, isValueLength := types.Get(src[offset])

	if typ == types.Str {
		var v string
		v, newOffset, err = ReadString(src, offset)
		return fast.StringToBytes(v), newOffset, err
	}

	if typ != types.Bin {
		err = expectedType(src[offset], types.Bin)
		return
	}

	offset++

	if !isValueLength {
		if offset+length > len(src) {
			return nil, offset, ErrShortBuffer
		}

		l := length
		length = intFromBuf[int](src[offset : offset+l])
		offset += l
	}

	if offset+length > len(src) {
		return nil, offset, ErrShortBuffer
	}

	data = src[offset : offset+length]
	offset += length

	return data, offset, nil
}

// AppendBinaryAppender appends a binary object to `dst` using a `BinaryAppender`.
// Returns the updated byte slice.
func AppendBinaryAppender(dst []byte, s internal.BinaryAppender) []byte {
	return AppendBinaryUnknownLength(dst, func(dst []byte) []byte {
		dst, _ = s.AppendBinary(dst)
		return dst
	})
}

// AppendBinaryUnknownLength appends a MessagePack binary header and binary data to the destination byte slice
// when the length of the binary data is unknown. It reserves space for the header, appends the data using the provided
// function `fn`, and updates the header with the actual length of the data.
func AppendBinaryUnknownLength(dst []byte, fn func(dst []byte) []byte) []byte {
	// We don't know the length of the binary, so assume the longest possible binary.
	start := len(dst)
	dst = append(dst, 0xc6, 0, 0, 0, 0)

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
