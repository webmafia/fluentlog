package msgpack

import (
	"errors"
	"fmt"
	"io"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

var (
	// ErrShortBuffer is returned when the byte slice is too short to read the expected data.
	ErrShortBuffer = io.ErrShortBuffer
	// ErrInvalidFormat is returned when the data does not conform to the expected MessagePack format.
	ErrInvalidFormat        = errors.New("invalid MessagePack format")
	ErrInvalidHeaderByte    = errors.New("invalid header byte")
	ErrInvalidExtByte       = errors.New("invalid extension byte")
	ErrReachedMaxBufferSize = errors.New("reached max buffer size")
)

func expectedType(c byte, expected types.Type) (err error) {
	return fmt.Errorf("%w: expected %s, got 0x%02x", ErrInvalidHeaderByte, expected, c)
}

func expectedTypes(c byte, expected ...types.Type) (err error) {
	return fmt.Errorf("%w: expected any of %v, got 0x%02x", ErrInvalidHeaderByte, expected, c)
}

func expectedExtType(got, expected byte) (err error) {
	return fmt.Errorf("%w: expected 0x%02x, got 0x%02x", ErrInvalidExtByte, expected, got)
}
