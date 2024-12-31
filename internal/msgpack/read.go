package msgpack

import (
	"encoding/binary"
	"io"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// Reads a MessagePack value from r, and appends it to dst. Returns the extended
// byte slice, the MessagePack type read, the number of succeeding subvalues (if
// array or map, otherwise it will always be zero), and any occurred error.
// Does only read exactly as many bytes as needed for the particular type.
func Read(dst []byte, r io.Reader) (b []byte, t types.Type, n int, err error) {
	var firstByte [1]byte
	if _, err = io.ReadFull(r, firstByte[:]); err != nil {
		return dst, t, n, err
	}
	typeByte := firstByte[0]
	t, length, isValueLength := types.Get(typeByte)
	dst = append(dst, typeByte) // Append the first byte

	if t == types.Array || t == types.Map {
		// Handle compound types by reading only the header to determine n
		if isValueLength {
			n = length // Embedded length directly represents the number of subvalues
		} else {
			var lengthBuf [4]byte
			var headerLength int
			switch typeByte {
			case 0xdc, 0xde: // array16, map16
				headerLength = 2
			case 0xdd, 0xdf: // array32, map32
				headerLength = 4
			}

			if headerLength > 0 {
				if _, err = io.ReadFull(r, lengthBuf[:headerLength]); err != nil {
					return dst, t, n, err
				}
				dst = append(dst, lengthBuf[:headerLength]...)
				switch headerLength {
				case 2:
					n = int(binary.BigEndian.Uint16(lengthBuf[:2]))
				case 4:
					n = int(binary.BigEndian.Uint32(lengthBuf[:4]))
				}
			}
		}
	} else {
		// Handle fixed-length or variable-length types
		if !isValueLength {
			// Read the length field for variable-length types
			var lengthBuf [4]byte
			var headerLength int
			switch typeByte {
			case 0xd9, 0xc4: // str8, bin8
				headerLength = 1
			case 0xda, 0xc5: // str16, bin16
				headerLength = 2
			case 0xdb, 0xc6: // str32, bin32
				headerLength = 4
			}

			if headerLength > 0 {
				if _, err = io.ReadFull(r, lengthBuf[:headerLength]); err != nil {
					return dst, t, n, err
				}
				dst = append(dst, lengthBuf[:headerLength]...)
				switch headerLength {
				case 1:
					length = int(lengthBuf[0])
				case 2:
					length = int(binary.BigEndian.Uint16(lengthBuf[:2]))
				case 4:
					length = int(binary.BigEndian.Uint32(lengthBuf[:4]))
				}
			}
		}

		if length > 0 {
			// Read the value data
			valueBuf := make([]byte, length)
			if _, err = io.ReadFull(r, valueBuf); err != nil {
				return dst, t, n, err
			}
			dst = append(dst, valueBuf...)
		}
	}

	return dst, t, n, nil
}
