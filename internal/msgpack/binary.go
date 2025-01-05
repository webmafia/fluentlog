package msgpack

import (
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
		return nil, offset, expectedType(b, types.Bin)
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
	// Reserve space for the header, assuming the shortest possible binary type initially.
	start := len(dst)
	dst = append(dst, 0xc4, 0) // Placeholder for bin8 header

	sizeFrom := len(dst)
	dst = fn(dst)
	sizeTo := len(dst)
	l := sizeTo - sizeFrom

	// Update the header based on the actual length of the binary data.
	switch {
	case l <= 0xFF:
		dst[start] = 0xc4
		dst[start+1] = byte(l)
	case l <= 0xFFFF:
		dst[start] = 0xc5
		dst = append(dst[:start+1], append([]byte{byte(l >> 8), byte(l)}, dst[start+2:]...)...)
	default:
		dst[start] = 0xc6
		dst = append(dst[:start+1], append([]byte{byte(l >> 24), byte(l >> 16), byte(l >> 8), byte(l)}, dst[start+2:]...)...)
	}

	return dst
}
