package msgpack

import "github.com/webmafia/fluentlog/internal/msgpack/types"

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
