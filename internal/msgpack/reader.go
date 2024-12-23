package msgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// Reader reads MessagePack-encoded data from an io.Reader.
// It buffers everything until explicitly released.
// The buffer is not altered until explicitly released, allowing for multiple messages to be read with the same Reader.
type Reader struct {
	b *buffer.Buffer
}

// NewReader creates a new Reader with the provided io.Reader and buffer.
// The buffer is passed in to allow for reuse and to avoid allocations.
func NewReader(b *buffer.Buffer) Reader {
	return Reader{
		b: b,
	}
}

// func (r *Reader) Reset(reader io.Reader) {
// 	r.r = reader
// 	r.buf = r.buf[:0]
// 	r.size = 0
// 	r.pos = 0
// }

// func (r *Reader) Rewind() {
// 	r.b.SetPos(0)
// }

// func (r *Reader) Pos() int {
// 	return r.b.Pos()
// }

// func (r *Reader) ResetToPos(pos int) {
// 	r.pos = min(pos, r.size)
// 	r.size = r.pos
// }

// func (r *Reader) ConsumedBuffer() []byte {
// 	return r.buf[:r.pos]
// }

// // Change reader without touching current buffer
// func (r *Reader) ChangeReader(reader io.Reader) {
// 	r.r = reader
// }

// PeekType peeks at the next MessagePack type without consuming any data.
func (r *Reader) PeekType() (t Type, err error) {
	c, err := r.b.PeekByte()

	if err != nil {
		return
	}

	switch {
	// Nil
	case c == 0xc0:
		return TypeNil, nil
	// Bool
	case c == 0xc2 || c == 0xc3:
		return TypeBool, nil
	// Positive FixInt
	case c <= 0x7f:
		return TypeInt, nil
	// Negative FixInt
	case c >= 0xe0:
		return TypeInt, nil
	// Int
	case c >= 0xd0 && c <= 0xd3:
		return TypeInt, nil
	// Uint
	case c >= 0xcc && c <= 0xcf:
		return TypeUint, nil
	// Float32
	case c == 0xca:
		return TypeFloat32, nil
	// Float64
	case c == 0xcb:
		return TypeFloat64, nil
	// FixStr
	case c >= 0xa0 && c <= 0xbf:
		return TypeString, nil
	// Str8, Str16, Str32
	case c >= 0xd9 && c <= 0xdb:
		return TypeString, nil
	// Bin8, Bin16, Bin32
	case c >= 0xc4 && c <= 0xc6:
		return TypeBinary, nil
	// FixArray, Array16, Array32
	case c >= 0x90 && c <= 0x9f || c == 0xdc || c == 0xdd:
		return TypeArray, nil
	// FixMap, Map16, Map32
	case c >= 0x80 && c <= 0x8f || c == 0xde || c == 0xdf:
		return TypeMap, nil
	// Ext types (including Timestamp)
	case c >= 0xd4 && c <= 0xd8 || c >= 0xc7 && c <= 0xc9:
		// We need to read the ext type to determine if it's a Timestamp
		extType, err := r.peekExtType(c)
		if err != nil {
			return TypeExt, nil // If we can't determine, default to TypeExt
		}
		if extType == -1 || extType == 0x00 {
			return TypeTimestamp, nil
		}
		return TypeExt, nil
	default:
		return TypeUnknown, fmt.Errorf("unknown MessagePack type: 0x%02x", c)
	}
}

// peekExtType peeks at the ext type without consuming any data.
// This is a helper function for PeekType.
func (r *Reader) peekExtType(c byte) (t int8, err error) {
	var headerSize int

	switch c {
	case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
		headerSize = 2
	case 0xc7:
		headerSize = 3
	case 0xc8:
		headerSize = 4
	case 0xc9:
		headerSize = 6
	default:
		return 0, fmt.Errorf("invalid ext header byte: 0x%02x", c)
	}

	header, err := r.b.PeekBytes(headerSize)

	if err != nil {
		return
	}

	var extType int8
	switch headerSize {
	case 2:
		extType = int8(header[1])
	case 3:
		extType = int8(header[2])
	case 4:
		extType = int8(header[3])
	case 6:
		extType = int8(header[5])
	}

	return extType, nil
}

// Release resets the reader state after processing a message.
// It discards consumed data and prepares for the next message.
func (r *Reader) Release() {
	r.b.Release()
}

func (r *Reader) ReleaseBefore(n int) {
	r.b.ReleaseBefore(n)
}

func (r *Reader) ReleaseAfter(n int) {
	r.b.ReleaseAfter(n)
}

func (r *Reader) readInt16() (i int, err error) {
	var b []byte

	if b, err = r.b.ReadBytes(2); err == nil {
		i = int(b[0])<<8 | int(b[1])
	}

	return
}

func (r *Reader) readInt32() (i int, err error) {
	var b []byte

	if b, err = r.b.ReadBytes(4); err == nil {
		i = int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	}

	return
}

// ReadArrayHeader reads an array header from the buffered data.
func (r *Reader) ReadArrayHeader() (length int, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var b []byte

	switch {

	case c >= 0x90 && c <= 0x9f:
		length = int(c & 0x0f)

	case c == 0xdc:
		length, err = r.readInt16()

	case c == 0xdd:
		length, err = r.readInt32()

	default:
		err = fmt.Errorf("invalid array header byte: 0x%02x", b)
	}

	if err != nil {
		r.b.AdjustPos(-1)
	}

	return
}

// ReadMapHeader reads a map header from the buffered data.
func (r *Reader) ReadMapHeader() (length int, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var b []byte

	switch {

	case c >= 0x80 && c <= 0x8f:
		length = int(c & 0x0f)

	case c == 0xde:
		length, err = r.readInt16()

	case c == 0xdf:
		length, err = r.readInt32()

	default:
		fmt.Errorf("invalid map header byte: 0x%02x", b)
	}

	if err != nil {
		r.b.AdjustPos(-1)
	}

	return
}

// ReadString reads a string from the buffered data.
func (r *Reader) ReadString() (str string, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var headLen int
	var length int

	switch {

	case c >= 0xa0 && c <= 0xbf:
		length = int(c & 0x1f)

	case c == 0xd9:
		var v uint8
		v, err = r.b.ReadByte()
		headLen = 1
		length = int(v)

	case c == 0xda:
		length, err = r.readInt16()
		headLen = 2

	case c == 0xdb:
		length, err = r.readInt32()
		headLen = 4

	default:
		err = fmt.Errorf("invalid string header byte: 0x%02x", c)
	}

	if err != nil {
		r.b.AdjustPos(-1)
	} else if str, err = r.b.ReadString(length); err != nil {
		r.b.AdjustPos(-(headLen + 1))
	}

	return
}

// ReadInt reads an integer from the buffered data.
func (r *Reader) ReadInt() (i int64, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var length int

	switch {
	case c <= 0x7f || c >= 0xe0:
		length = 0
	case c == 0xd0 || c == 0xcc:
		length = 1
	case c == 0xd1 || c == 0xcd:
		length = 2
	case c == 0xd2 || c == 0xce:
		length = 4
	case c == 0xd3 || c == 0xcf:
		length = 8
	default:
		err = fmt.Errorf("invalid int header byte: 0x%02x", c)
	}

	if err != nil {
		r.b.AdjustPos(-1)
		return
	}

	b, err := r.b.ReadBytes(length)

	if err != nil {
		r.b.AdjustPos(-1)
		return
	}

	switch length {

	case 0:
		i = int64(c)

	case 1:
		i = int64(b[0])

	case 2:
		i = int64(b[0])<<8 | int64(b[1])

	case 4:
		i = int64(b[0])<<24 | int64(b[1])<<16 | int64(b[2])<<8 | int64(b[3])

	case 8:
		i = int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
			int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
	}

	return
}

// ReadUint reads an unsigned integer from the buffered data.
func (r *Reader) ReadUint() (i uint64, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var length int

	switch {

	case c <= 0x7f:
		length = 0

	case c == 0xcc:
		length = 1

	case c == 0xcd:
		length = 2

	case c == 0xce:
		length = 4

	case c == 0xcf:
		length = 8

	default:
		return 0, fmt.Errorf("invalid uint header byte: 0x%02x", c)
	}

	if err != nil {
		r.b.AdjustPos(-1)
		return
	}

	b, err := r.b.ReadBytes(length)

	if err != nil {
		r.b.AdjustPos(-1)
		return
	}

	switch length {

	case 0:
		i = uint64(c)

	case 1:
		i = uint64(b[0])

	case 2:
		i = uint64(b[0])<<8 | uint64(b[1])

	case 4:
		i = uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])

	case 8:
		i = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}

	return
}

// ReadBool reads a boolean value from the buffered data.
func (r *Reader) ReadBool() (b bool, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	switch c {
	case 0xc2:
		return false, nil
	case 0xc3:
		return true, nil
	default:
		r.b.AdjustPos(-1)
		return false, fmt.Errorf("invalid bool header byte: 0x%02x", c)
	}
}

// ReadNil reads a nil value from the buffered data.
func (r *Reader) ReadNil() (err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	if c != 0xc0 {
		r.b.AdjustPos(-1)
		return fmt.Errorf("expected nil (0xc0), got 0x%02x", c)
	}

	return
}

// ReadBinary reads binary data from the buffered data.
func (r *Reader) ReadBinary() (b []byte, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var headLen int
	var length int

	switch c {

	case 0xc4:
		var v uint8
		v, err = r.b.ReadByte()
		headLen = 1
		length = int(v)

	case 0xc5:
		length, err = r.readInt16()
		headLen = 2

	case 0xc6:
		length, err = r.readInt32()
		headLen = 4

	default:
		r.b.AdjustPos(-1)

		// Check if it might be a string - if so, then we can read it as a byte slice
		str, err := r.ReadString()

		if err != nil {
			return nil, fmt.Errorf("invalid binary header byte: 0x%02x", b)
		}

		return internal.S2B(str), nil
	}

	if err != nil {
		r.b.AdjustPos(-1)
	} else if b, err = r.b.ReadBytes(length); err != nil {
		r.b.AdjustPos(-(headLen + 1))
	}

	return
}

// func (r *Reader) SkipBinaryHeader() (int, error) {
// 	if err := r.fill(1); err != nil {
// 		return 0, err
// 	}

// 	b := r.buf[r.pos]
// 	var length int
// 	var headerSize int

// 	switch b {
// 	case 0xc4: // bin8
// 		headerSize = 2
// 		if err := r.fill(headerSize); err != nil {
// 			return 0, err
// 		}
// 		length = int(r.buf[r.pos+1])
// 	case 0xc5: // bin16
// 		headerSize = 3
// 		if err := r.fill(headerSize); err != nil {
// 			return 0, err
// 		}
// 		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
// 	case 0xc6: // bin32
// 		headerSize = 5
// 		if err := r.fill(headerSize); err != nil {
// 			return 0, err
// 		}
// 		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
// 	default:
// 		return 0, fmt.Errorf("SkipBinaryHeader: expected binary header but got type 0x%02x", b)
// 	}

// 	// Consume only the header, not the binary data
// 	r.consume(headerSize)

// 	return length, nil
// }

// ReadFloat32 reads a 32-bit floating point number from the buffered data.
func (r *Reader) ReadFloat32() (f float32, err error) {
	b, err := r.b.ReadBytes(5)

	if err != nil {
		return
	}

	if b[0] != 0xca {
		r.b.AdjustPos(-5)
		return 0, fmt.Errorf("expected float32 (0xca), got 0x%02x", b[0])
	}

	bits := uint32(b[1])<<24 | uint32(b[2])<<16 | uint32(b[3])<<8 | uint32(b[4])

	return math.Float32frombits(bits), nil
}

// ReadFloat64 reads a 64-bit floating point number from the buffered data.
func (r *Reader) ReadFloat64() (f float64, err error) {
	b, err := r.b.ReadBytes(9)

	if err != nil {
		return
	}

	if b[0] != 0xcb {
		r.b.AdjustPos(-9)
		return 0, fmt.Errorf("expected float64 (0xcb), got 0x%02x", b[0])
	}

	bits := uint64(b[1])<<56 | uint64(b[2])<<48 | uint64(b[3])<<40 | uint64(b[4])<<32 |
		uint64(b[5])<<24 | uint64(b[6])<<16 | uint64(b[7])<<8 | uint64(b[8])

	return math.Float64frombits(bits), nil
}

// ReadTimestamp reads a timestamp value from the buffered data.
// It supports both EventTime (ext type 0) and integer timestamps.
func (r *Reader) ReadTimestamp() (t time.Time, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	var s, ns int64
	var b []byte

	switch c {

	case 0xd7: // fixext8
		if b, err = r.b.ReadBytes(9); err != nil {
			r.b.AdjustPos(-1)
			return
		}

		if b[0] != 0x00 {
			r.b.AdjustPos(-1)
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(b[1:5])))
		ns = int64(int32(binary.BigEndian.Uint32(b[5:9])))

	case 0xc7: // ext8
		if b, err = r.b.ReadBytes(10); err != nil {
			r.b.AdjustPos(-1)
			return
		}

		if b[0] != 0x08 || b[1] != 0x00 {
			r.b.AdjustPos(-1)
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(b[2:6])))
		ns = int64(int32(binary.BigEndian.Uint32(b[6:10])))

	default:
		r.b.AdjustPos(-1)

		if s, err = r.ReadInt(); err != nil {
			return
		}
	}

	return time.Unix(s, ns), nil
}

func (r *Reader) Skip() error {

}

func (r *Reader) readLen() (length int, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	// Get the length of the string or the length of the "length field"
	length, isValLen := types.GetLength(c)

	// If the length is encoded in additional bytes, decode it
	if !isValLen {
		encodedLength, err := r.b.ReadBytes(length)

		if err != nil {
			return 0, err
		}

		length = types.GetInt(encodedLength)
	}

	return
}

// func (r *Reader) ReadRaw() ([]byte, error) {
// 	startPos := r.pos

// 	// Skip over the next item
// 	err := r.Skip()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Return the bytes from startPos to r.pos
// 	rawBytes := r.buf[startPos:r.pos]
// 	r.consume(len(rawBytes))

// 	return rawBytes, nil
// }

// func (r *Reader) Skip() error {
// 	if err := r.fill(1); err != nil {
// 		return err
// 	}
// 	b := r.buf[r.pos]

// 	switch {
// 	// Nil
// 	case b == 0xc0:
// 		r.consume(1)
// 		return nil
// 	// Bool
// 	case b == 0xc2 || b == 0xc3:
// 		r.consume(1)
// 		return nil
// 	// Positive FixInt and Negative FixInt
// 	case b <= 0x7f || b >= 0xe0:
// 		r.consume(1)
// 		return nil
// 	// Int8, Int16, Int32, Int64
// 	case b == 0xd0:
// 		return r.consumeN(2)
// 	case b == 0xd1:
// 		return r.consumeN(3)
// 	case b == 0xd2:
// 		return r.consumeN(5)
// 	case b == 0xd3:
// 		return r.consumeN(9)
// 	// Uint8, Uint16, Uint32, Uint64
// 	case b == 0xcc:
// 		return r.consumeN(2)
// 	case b == 0xcd:
// 		return r.consumeN(3)
// 	case b == 0xce:
// 		return r.consumeN(5)
// 	case b == 0xcf:
// 		return r.consumeN(9)
// 	// Float32
// 	case b == 0xca:
// 		return r.consumeN(5)
// 	// Float64
// 	case b == 0xcb:
// 		return r.consumeN(9)
// 	// FixStr
// 	case b >= 0xa0 && b <= 0xbf:
// 		length := int(b & 0x1f)
// 		return r.consumeN(1 + length)
// 	// Str8
// 	case b == 0xd9:
// 		if err := r.fill(2); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])
// 		return r.consumeN(2 + length)
// 	// Str16
// 	case b == 0xda:
// 		if err := r.fill(3); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
// 		return r.consumeN(3 + length)
// 	// Str32
// 	case b == 0xdb:
// 		if err := r.fill(5); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
// 		return r.consumeN(5 + length)
// 	// Bin8
// 	case b == 0xc4:
// 		if err := r.fill(2); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])
// 		return r.consumeN(2 + length)
// 	// Bin16
// 	case b == 0xc5:
// 		if err := r.fill(3); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
// 		return r.consumeN(3 + length)
// 	// Bin32
// 	case b == 0xc6:
// 		if err := r.fill(5); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
// 		return r.consumeN(5 + length)
// 	// FixArray
// 	case b >= 0x90 && b <= 0x9f:
// 		length := int(b & 0x0f)
// 		r.consume(1)
// 		return r.skipNItems(length)
// 	// Array16
// 	case b == 0xdc:
// 		if err := r.fill(3); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
// 		r.consume(3)
// 		return r.skipNItems(length)
// 	// Array32
// 	case b == 0xdd:
// 		if err := r.fill(5); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
// 		r.consume(5)
// 		return r.skipNItems(length)
// 	// FixMap
// 	case b >= 0x80 && b <= 0x8f:
// 		length := int(b & 0x0f)
// 		r.consume(1)
// 		return r.skipNMapItems(length)
// 	// Map16
// 	case b == 0xde:
// 		if err := r.fill(3); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
// 		r.consume(3)
// 		return r.skipNMapItems(length)
// 	// Map32
// 	case b == 0xdf:
// 		if err := r.fill(5); err != nil {
// 			return err
// 		}
// 		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
// 		r.consume(5)
// 		return r.skipNMapItems(length)
// 	// FixExt
// 	case b >= 0xd4 && b <= 0xd8:
// 		dataSize := 1 << (b - 0xd4)
// 		return r.consumeN(2 + dataSize)
// 	// Ext8
// 	case b == 0xc7:
// 		if err := r.fill(2); err != nil {
// 			return err
// 		}
// 		dataSize := int(r.buf[r.pos+1])
// 		return r.consumeN(3 + dataSize)
// 	// Ext16
// 	case b == 0xc8:
// 		if err := r.fill(3); err != nil {
// 			return err
// 		}
// 		dataSize := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
// 		return r.consumeN(4 + dataSize)
// 	// Ext32
// 	case b == 0xc9:
// 		if err := r.fill(5); err != nil {
// 			return err
// 		}
// 		dataSize := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
// 		return r.consumeN(6 + dataSize)
// 	default:
// 		return fmt.Errorf("unsupported type: 0x%02x", b)
// 	}
// }

// // consumeN advances the read position by `n` bytes, ensuring enough data is available.
// func (r *Reader) consumeN(n int) error {
// 	if err := r.fill(n); err != nil {
// 		return err
// 	}
// 	r.consume(n)
// 	return nil
// }

// // skipNItems skips over `n` MessagePack items.
// func (r *Reader) skipNItems(n int) error {
// 	for i := 0; i < n; i++ {
// 		if err := r.Skip(); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// // skipNMapItems skips over `n` key-value pairs in a map.
// func (r *Reader) skipNMapItems(n int) error {
// 	for i := 0; i < n; i++ {
// 		// Skip key
// 		if err := r.Skip(); err != nil {
// 			return err
// 		}
// 		// Skip value
// 		if err := r.Skip(); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (r *Reader) SkipMap() error {
// 	// Read the map header to determine the number of key-value pairs
// 	mapLen, err := r.ReadMapHeader()
// 	if err != nil {
// 		return fmt.Errorf("SkipMap: failed to read map header: %v", err)
// 	}

// 	// Skip all keys and values
// 	return r.skipNMapItems(mapLen)
// }

// func (r *Reader) PeekBytes(n int) ([]byte, error) {
// 	if err := r.fill(n); err != nil {
// 		return nil, err
// 	}

// 	return r.buf[r.pos : r.pos+n], nil
// }

// var _ io.Reader = (*Reader)(nil)

// func (r *Reader) Read(p []byte) (int, error) {
// 	// If there's no unconsumed data in the buffer, try to fill it.
// 	if r.pos >= r.size {
// 		if err := r.fill(1); err != nil {
// 			// Return EOF if no data remains and it's not an unexpected error
// 			if err == io.EOF {
// 				return 0, io.EOF
// 			}
// 			return 0, err
// 		}
// 	}

// 	// Determine how much data to copy
// 	n := copy(p, r.buf[r.pos:r.size])
// 	r.pos += n // Update the position in the buffer

// 	return n, nil
// }
