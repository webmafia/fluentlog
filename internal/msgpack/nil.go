package msgpack

import "fmt"

// AppendNil appends a nil value to `dst` as a MessagePack-encoded value.
// Returns the updated byte slice.
func AppendNil(dst []byte) []byte {
	return append(dst, 0xc0)
}

// ReadNil reads a nil value from `src` starting at `offset`.
// Returns the new offset and an error if the value is not nil or the buffer is too short.
func ReadNil(src []byte, offset int) (newOffset int, err error) {
	if offset >= len(src) {
		return offset, ErrShortBuffer
	}
	if src[offset] != 0xc0 {
		return offset, fmt.Errorf("expected nil (0xc0), got 0x%02x", src[offset])
	}
	return offset + 1, nil
}
