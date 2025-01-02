package msgpack

import (
	"io"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// Reads a MessagePack value from r, and appends it to dst. Returns the extended
// byte slice, the MessagePack type read, the number of succeeding subvalues (if
// array or map, otherwise it will always be zero), and any occurred error.
// Does only read exactly as many bytes as needed for the particular type.
func Read(dst []byte, r io.Reader) (b []byte, t types.Type, n int, err error) {
	var firstByte []byte

	dst, firstByte = appendBuf(dst, 1)

	if _, err = io.ReadFull(r, firstByte[:]); err != nil {
		return
	}

	typeByte := firstByte[0]
	t, length, isValueLength := types.Get(typeByte)
	// dst = append(dst, typeByte) // Append the first byte

	if !isValueLength {
		var lengthBuf []byte

		dst, lengthBuf = appendBuf(dst, length)

		if _, err = io.ReadFull(r, lengthBuf); err != nil {
			return
		}

		// dst = append(dst, lengthBuf[:length]...)
		length = intFromBuf[int](lengthBuf)
	}

	if t == types.Array || t == types.Map {
		n = length
	} else {
		var buf []byte
		dst, buf = appendBuf(dst, length)

		if _, err = io.ReadFull(r, buf); err != nil {
			return
		}
	}

	return dst, t, n, nil
}

func appendBuf(dst []byte, n int) (newSlice []byte, buf []byte) {
	l := len(dst)

	if tot := l + n; tot > cap(dst) {
		newSlice = fast.MakeNoZeroCap(tot, tot+64)
		copy(newSlice, dst)
	} else {
		newSlice = dst[:tot]
	}

	buf = newSlice[l:]
	return
}
