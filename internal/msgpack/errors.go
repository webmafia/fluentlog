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
	ErrInvalidFormat = errors.New("invalid MessagePack format")
)

func expectedType(c byte, expected types.Type) (err error) {
	return fmt.Errorf("invalid %s header byte: 0x%02x", expected, c)
}
