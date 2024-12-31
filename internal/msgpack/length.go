package msgpack

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func getLength(c byte, fn func(l int) ([]byte, error)) (typ types.Type, length int, err error) {
	typ, length, isValueLength := types.Get(c)

	if length > 0 && !isValueLength {
		length, err = getEncodedLength(length, fn)
	}

	return
}

func getLengthWithHead(c byte, fn func(l int) ([]byte, error)) (typ types.Type, headLength int, bodyLength int, err error) {
	typ, bodyLength, isValueLength := types.Get(c)

	headLength += 1

	if bodyLength > 0 && !isValueLength {
		headLength += bodyLength
		bodyLength, err = getEncodedLength(bodyLength, fn)
	}

	return
}

func getLengthFromBuf(buf []byte) (typ types.Type, headLength int, bodyLength int, err error) {
	typ, bodyLength, isValueLength := types.Get(buf[0])

	buf = buf[1:]
	headLength += 1

	if bodyLength > 0 && !isValueLength {
		if bodyLength > len(buf) {
			err = ErrShortBuffer
			return
		}

		headLength += bodyLength

		switch len(buf) {

		case 1:
			bodyLength = int(buf[0])

		case 2:
			bodyLength = int(binary.BigEndian.Uint16(buf))

		case 4:
			bodyLength = int(binary.BigEndian.Uint32(buf))

		case 8:
			bodyLength = int(binary.BigEndian.Uint64(buf))

		default:
			err = errors.New("invalid length")
		}
	}

	return
}

func getEncodedLength(length int, fn func(l int) ([]byte, error)) (int, error) {
	buf, err := fn(length)

	if err != nil {
		return 0, err
	}

	if len(buf) != length {
		return 0, fmt.Errorf("expected %d bytes, got %d bytes", length, len(buf))
	}

	switch length {

	case 1:
		return int(buf[0]), nil

	case 2:
		return int(binary.BigEndian.Uint16(buf)), nil

	case 4:
		return int(binary.BigEndian.Uint32(buf)), nil

	case 8:
		return int(binary.BigEndian.Uint64(buf)), nil

	}

	return 0, errors.New("invalid length")
}

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func intFromBuf[T numeric](b []byte) T {
	switch len(b) {

	case 1:
		return T(b[0])

	case 2:
		return T(binary.BigEndian.Uint16(b))

	case 4:
		return T(binary.BigEndian.Uint32(b))

	case 8:
		return T(binary.BigEndian.Uint64(b))
	}

	return 0
}
