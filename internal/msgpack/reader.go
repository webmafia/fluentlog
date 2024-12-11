package msgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/webmafia/fluentlog/internal"
)

// Reader reads MessagePack-encoded data from an io.Reader.
// It buffers everything until explicitly released.
// The buffer is not altered until explicitly released, allowing for multiple messages to be read with the same Reader.
type Reader struct {
	r    io.Reader
	buf  []byte
	size int // Total number of valid bytes in `buf`
	pos  int // Current read position in `buf`
}

// NewReader creates a new Reader with the provided io.Reader and buffer.
// The buffer is passed in to allow for reuse and to avoid allocations.
func NewReader(r io.Reader, buffer []byte) Reader {
	return Reader{
		r:   r,
		buf: buffer[:0], // Initialize buffer with zero length but existing capacity.
	}
}

func (r *Reader) Reset(reader io.Reader) {
	r.r = reader
	r.buf = r.buf[:0]
	r.size = 0
	r.pos = 0
}

func (r *Reader) Rewind() {
	r.pos = 0
}

func (r *Reader) Pos() int {
	return r.pos
}

func (r *Reader) ResetToPos(pos int) {
	r.pos = min(pos, r.size)
	r.size = r.pos
}

func (r *Reader) ConsumedBuffer() []byte {
	return r.buf[:r.pos]
}

// Change reader without touching current buffer
func (r *Reader) ChangeReader(reader io.Reader) {
	r.r = reader
}

// PeekType peeks at the next MessagePack type without consuming any data.
func (r *Reader) PeekType() (Type, error) {
	if err := r.fill(1); err != nil {
		return TypeUnknown, err
	}
	b := r.buf[r.pos]

	switch {
	// Nil
	case b == 0xc0:
		return TypeNil, nil
	// Bool
	case b == 0xc2 || b == 0xc3:
		return TypeBool, nil
	// Positive FixInt
	case b <= 0x7f:
		return TypeInt, nil
	// Negative FixInt
	case b >= 0xe0:
		return TypeInt, nil
	// Int
	case b >= 0xd0 && b <= 0xd3:
		return TypeInt, nil
	// Uint
	case b >= 0xcc && b <= 0xcf:
		return TypeUint, nil
	// Float32
	case b == 0xca:
		return TypeFloat32, nil
	// Float64
	case b == 0xcb:
		return TypeFloat64, nil
	// FixStr
	case b >= 0xa0 && b <= 0xbf:
		return TypeString, nil
	// Str8, Str16, Str32
	case b >= 0xd9 && b <= 0xdb:
		return TypeString, nil
	// Bin8, Bin16, Bin32
	case b >= 0xc4 && b <= 0xc6:
		return TypeBinary, nil
	// FixArray, Array16, Array32
	case b >= 0x90 && b <= 0x9f || b == 0xdc || b == 0xdd:
		return TypeArray, nil
	// FixMap, Map16, Map32
	case b >= 0x80 && b <= 0x8f || b == 0xde || b == 0xdf:
		return TypeMap, nil
	// Ext types (including Timestamp)
	case b >= 0xd4 && b <= 0xd8 || b >= 0xc7 && b <= 0xc9:
		// We need to read the ext type to determine if it's a Timestamp
		extType, err := r.peekExtType()
		if err != nil {
			return TypeExt, nil // If we can't determine, default to TypeExt
		}
		if extType == -1 || extType == 0x00 {
			return TypeTimestamp, nil
		}
		return TypeExt, nil
	default:
		return TypeUnknown, fmt.Errorf("unknown MessagePack type: 0x%02x", b)
	}
}

// peekExtType peeks at the ext type without consuming any data.
// This is a helper function for PeekType.
func (r *Reader) peekExtType() (int8, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}
	b := r.buf[r.pos]
	var headerSize int

	switch b {
	case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
		headerSize = 2
	case 0xc7:
		headerSize = 3
	case 0xc8:
		headerSize = 4
	case 0xc9:
		headerSize = 6
	default:
		return 0, fmt.Errorf("invalid ext header byte: 0x%02x", b)
	}

	if err := r.fill(headerSize); err != nil {
		return 0, err
	}

	var extType int8
	switch headerSize {
	case 2:
		extType = int8(r.buf[r.pos+1])
	case 3:
		extType = int8(r.buf[r.pos+2])
	case 4:
		extType = int8(r.buf[r.pos+3])
	case 6:
		extType = int8(r.buf[r.pos+5])
	}

	return extType, nil
}

// fill ensures that the buffer contains at least `n` bytes from the current position.
func (r *Reader) fill(n int) error {
	required := n - (r.size - r.pos)
	if required <= 0 {
		return nil
	}

	// Ensure the buffer has enough capacity.
	if cap(r.buf) < r.size+required {
		newBuf := make([]byte, r.size+required)
		copy(newBuf, r.buf[:r.size])
		r.buf = newBuf
	} else {
		// Adjust the buffer slice to include the new capacity.
		r.buf = r.buf[:cap(r.buf)]
	}

	// Read data into the buffer.
	nRead, err := io.ReadAtLeast(r.r, r.buf[r.size:], required)
	r.size += nRead

	if err != nil {
		return err
	}

	return nil
}

// consume advances the read position by `n` bytes.
func (r *Reader) consume(n int) {
	r.pos += n
}

// Release resets the reader state after processing a message.
// It discards consumed data and prepares for the next message.
func (r *Reader) Release() {
	// Keep unconsumed data by slicing the buffer.
	copy(r.buf, r.buf[r.pos:r.size])
	r.size -= r.pos
	r.pos = 0
}

func (r *Reader) ReleaseTo(pos int) error {
	if pos > r.size || pos < 0 {
		return fmt.Errorf("ReleaseTo: invalid position %d (must be between 0 and %d)", pos, r.size)
	}

	// Calculate the size of unconsumed data
	unconsumed := r.size - r.pos

	// Move unconsumed data to position `pos`
	copy(r.buf[pos:], r.buf[r.pos:r.size])

	// Update buffer size to include all data up to `pos` and the unconsumed part
	r.size = pos + unconsumed

	// Reset the read position to `pos`
	r.pos = pos

	return nil
}

// ReadArrayHeader reads an array header from the buffered data.
func (r *Reader) ReadArrayHeader() (int, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	b := r.buf[r.pos]
	var length int
	var headerSize int

	switch {
	case b >= 0x90 && b <= 0x9f:
		length = int(b & 0x0f)
		headerSize = 1
	case b == 0xdc:
		headerSize = 3
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
	case b == 0xdd:
		headerSize = 5
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
	default:
		return 0, fmt.Errorf("invalid array header byte: 0x%02x", b)
	}

	r.consume(headerSize)
	return length, nil
}

// ReadMapHeader reads a map header from the buffered data.
func (r *Reader) ReadMapHeader() (int, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	b := r.buf[r.pos]
	var length int
	var headerSize int

	switch {
	case b >= 0x80 && b <= 0x8f:
		length = int(b & 0x0f)
		headerSize = 1
	case b == 0xde:
		headerSize = 3
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
	case b == 0xdf:
		headerSize = 5
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
	default:
		return 0, fmt.Errorf("invalid map header byte: 0x%02x", b)
	}

	r.consume(headerSize)
	return length, nil
}

// ReadString reads a string from the buffered data.
func (r *Reader) ReadString() (string, error) {
	if err := r.fill(1); err != nil {
		return "", err
	}
	b := r.buf[r.pos]
	var length int
	var headerSize int

	switch {
	case b >= 0xa0 && b <= 0xbf:
		length = int(b & 0x1f)
		headerSize = 1
	case b == 0xd9:
		headerSize = 2
		if err := r.fill(headerSize); err != nil {
			return "", err
		}
		length = int(r.buf[r.pos+1])
	case b == 0xda:
		headerSize = 3
		if err := r.fill(headerSize); err != nil {
			return "", err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
	case b == 0xdb:
		headerSize = 5
		if err := r.fill(headerSize); err != nil {
			return "", err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
	default:
		return "", fmt.Errorf("invalid string header byte: 0x%02x", b)
	}

	totalLength := headerSize + length
	if err := r.fill(totalLength); err != nil {
		return "", err
	}

	s := internal.B2S(r.buf[r.pos+headerSize : r.pos+totalLength])
	r.consume(totalLength)
	return s, nil
}

// ReadInt reads an integer from the buffered data.
func (r *Reader) ReadInt() (int64, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}
	b := r.buf[r.pos]
	var required int
	switch {
	case b <= 0x7f || b >= 0xe0:
		required = 1
	case b == 0xd0 || b == 0xcc:
		required = 2
	case b == 0xd1 || b == 0xcd:
		required = 3
	case b == 0xd2 || b == 0xce:
		required = 5
	case b == 0xd3 || b == 0xcf:
		required = 9
	default:
		return 0, fmt.Errorf("invalid int header byte: 0x%02x", b)
	}
	if err := r.fill(required); err != nil {
		return 0, err
	}

	var value int64
	switch {
	case b <= 0x7f:
		// positive fixint
		value = int64(b)
		r.consume(1)
	case b >= 0xe0:
		// negative fixint
		value = int64(int8(b))
		r.consume(1)
	case b == 0xd0:
		value = int64(int8(r.buf[r.pos+1]))
		r.consume(2)
	case b == 0xd1:
		value = int64(int16(r.buf[r.pos+1])<<8 | int16(r.buf[r.pos+2]))
		r.consume(3)
	case b == 0xd2:
		value = int64(int32(r.buf[r.pos+1])<<24 | int32(r.buf[r.pos+2])<<16 | int32(r.buf[r.pos+3])<<8 | int32(r.buf[r.pos+4]))
		r.consume(5)
	case b == 0xd3:
		value = int64(uint64(r.buf[r.pos+1])<<56 | uint64(r.buf[r.pos+2])<<48 | uint64(r.buf[r.pos+3])<<40 | uint64(r.buf[r.pos+4])<<32 |
			uint64(r.buf[r.pos+5])<<24 | uint64(r.buf[r.pos+6])<<16 | uint64(r.buf[r.pos+7])<<8 | uint64(r.buf[r.pos+8]))
		r.consume(9)
	case b == 0xcc:
		value = int64(r.buf[r.pos+1])
		r.consume(2)
	case b == 0xcd:
		value = int64(uint16(r.buf[r.pos+1])<<8 | uint16(r.buf[r.pos+2]))
		r.consume(3)
	case b == 0xce:
		value = int64(uint32(r.buf[r.pos+1])<<24 | uint32(r.buf[r.pos+2])<<16 | uint32(r.buf[r.pos+3])<<8 | uint32(r.buf[r.pos+4]))
		r.consume(5)
	case b == 0xcf:
		value = int64(uint64(r.buf[r.pos+1])<<56 | uint64(r.buf[r.pos+2])<<48 | uint64(r.buf[r.pos+3])<<40 | uint64(r.buf[r.pos+4])<<32 |
			uint64(r.buf[r.pos+5])<<24 | uint64(r.buf[r.pos+6])<<16 | uint64(r.buf[r.pos+7])<<8 | uint64(r.buf[r.pos+8]))
		r.consume(9)
	}
	return value, nil
}

// ReadUint reads an unsigned integer from the buffered data.
func (r *Reader) ReadUint() (uint64, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}
	b := r.buf[r.pos]
	var required int
	switch {
	case b <= 0x7f:
		required = 1
	case b == 0xcc:
		required = 2
	case b == 0xcd:
		required = 3
	case b == 0xce:
		required = 5
	case b == 0xcf:
		required = 9
	default:
		return 0, fmt.Errorf("invalid uint header byte: 0x%02x", b)
	}
	if err := r.fill(required); err != nil {
		return 0, err
	}

	var value uint64
	switch {
	case b <= 0x7f:
		value = uint64(b)
		r.consume(1)
	case b == 0xcc:
		value = uint64(r.buf[r.pos+1])
		r.consume(2)
	case b == 0xcd:
		value = uint64(r.buf[r.pos+1])<<8 | uint64(r.buf[r.pos+2])
		r.consume(3)
	case b == 0xce:
		value = uint64(r.buf[r.pos+1])<<24 | uint64(r.buf[r.pos+2])<<16 | uint64(r.buf[r.pos+3])<<8 | uint64(r.buf[r.pos+4])
		r.consume(5)
	case b == 0xcf:
		value = uint64(r.buf[r.pos+1])<<56 | uint64(r.buf[r.pos+2])<<48 | uint64(r.buf[r.pos+3])<<40 | uint64(r.buf[r.pos+4])<<32 |
			uint64(r.buf[r.pos+5])<<24 | uint64(r.buf[r.pos+6])<<16 | uint64(r.buf[r.pos+7])<<8 | uint64(r.buf[r.pos+8])
		r.consume(9)
	}
	return value, nil
}

// ReadBool reads a boolean value from the buffered data.
func (r *Reader) ReadBool() (bool, error) {
	if err := r.fill(1); err != nil {
		return false, err
	}
	b := r.buf[r.pos]
	switch b {
	case 0xc2:
		r.consume(1)
		return false, nil
	case 0xc3:
		r.consume(1)
		return true, nil
	default:
		return false, fmt.Errorf("invalid bool header byte: 0x%02x", b)
	}
}

// ReadNil reads a nil value from the buffered data.
func (r *Reader) ReadNil() error {
	if err := r.fill(1); err != nil {
		return err
	}
	if r.buf[r.pos] != 0xc0 {
		return fmt.Errorf("expected nil (0xc0), got 0x%02x", r.buf[r.pos])
	}
	r.consume(1)
	return nil
}

// ReadBinary reads binary data from the buffered data.
func (r *Reader) ReadBinary() ([]byte, error) {
	if err := r.fill(1); err != nil {
		return nil, err
	}
	b := r.buf[r.pos]
	var length int
	var headerSize int

	switch b {
	case 0xc4:
		headerSize = 2
		if err := r.fill(headerSize); err != nil {
			return nil, err
		}
		length = int(r.buf[r.pos+1])
	case 0xc5:
		headerSize = 3
		if err := r.fill(headerSize); err != nil {
			return nil, err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
	case 0xc6:
		headerSize = 5
		if err := r.fill(headerSize); err != nil {
			return nil, err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
	default:

		// Check if it might be a string - if so, then we can read it as a byte slice
		str, err := r.ReadString()

		if err != nil {
			return nil, fmt.Errorf("invalid binary header byte: 0x%02x", b)
		}

		return internal.S2B(str), nil
	}

	totalLength := headerSize + length
	if err := r.fill(totalLength); err != nil {
		return nil, err
	}

	data := r.buf[r.pos+headerSize : r.pos+totalLength]
	r.consume(totalLength)
	return data, nil
}

func (r *Reader) SkipBinaryHeader() (int, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	b := r.buf[r.pos]
	var length int
	var headerSize int

	switch b {
	case 0xc4: // bin8
		headerSize = 2
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])
	case 0xc5: // bin16
		headerSize = 3
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
	case 0xc6: // bin32
		headerSize = 5
		if err := r.fill(headerSize); err != nil {
			return 0, err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
	default:
		return 0, fmt.Errorf("SkipBinaryHeader: expected binary header but got type 0x%02x", b)
	}

	// Consume only the header, not the binary data
	r.consume(headerSize)

	return length, nil
}

// ReadFloat32 reads a 32-bit floating point number from the buffered data.
func (r *Reader) ReadFloat32() (float32, error) {
	if err := r.fill(5); err != nil {
		return 0, err
	}
	if r.buf[r.pos] != 0xca {
		return 0, fmt.Errorf("expected float32 (0xca), got 0x%02x", r.buf[r.pos])
	}
	bits := uint32(r.buf[r.pos+1])<<24 | uint32(r.buf[r.pos+2])<<16 | uint32(r.buf[r.pos+3])<<8 | uint32(r.buf[r.pos+4])
	value := math.Float32frombits(bits)
	r.consume(5)
	return value, nil
}

// ReadFloat64 reads a 64-bit floating point number from the buffered data.
func (r *Reader) ReadFloat64() (float64, error) {
	if err := r.fill(9); err != nil {
		return 0, err
	}
	if r.buf[r.pos] != 0xcb {
		return 0, fmt.Errorf("expected float64 (0xcb), got 0x%02x", r.buf[r.pos])
	}
	bits := uint64(r.buf[r.pos+1])<<56 | uint64(r.buf[r.pos+2])<<48 | uint64(r.buf[r.pos+3])<<40 | uint64(r.buf[r.pos+4])<<32 |
		uint64(r.buf[r.pos+5])<<24 | uint64(r.buf[r.pos+6])<<16 | uint64(r.buf[r.pos+7])<<8 | uint64(r.buf[r.pos+8])
	value := math.Float64frombits(bits)
	r.consume(9)
	return value, nil
}

// ReadTimestamp reads a timestamp value from the buffered data.
// It supports both EventTime (ext type 0) and integer timestamps.
func (r *Reader) ReadTimestamp() (t time.Time, err error) {
	if err = r.fill(1); err != nil {
		return
	}

	b := r.buf[r.pos]

	var s, ns int64

	switch b {

	case 0xd7: // fixext8
		if err = r.fill(10); err != nil {
			return
		}

		if r.buf[r.pos+1] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(r.buf[r.pos+2 : r.pos+6])))
		ns = int64(int32(binary.BigEndian.Uint32(r.buf[r.pos+6 : r.pos+10])))
		r.consume(10)

	case 0xc7: // ext8
		if err = r.fill(11); err != nil {
			return
		}

		if r.buf[r.pos+1] != 0x08 || r.buf[r.pos+2] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(r.buf[r.pos+3 : r.pos+7])))
		ns = int64(int32(binary.BigEndian.Uint32(r.buf[r.pos+7 : r.pos+11])))
		r.consume(11)

	default:
		if s, err = r.ReadInt(); err != nil {
			return
		}
	}

	return time.Unix(s, ns), nil
}

// ReadExt reads an extension object from the buffered data.
func (r *Reader) ReadExt() (int8, []byte, error) {
	if err := r.fill(1); err != nil {
		return 0, nil, err
	}
	b := r.buf[r.pos]
	var length int
	var headerSize int

	switch b {
	case 0xd4:
		headerSize = 2
		length = 1
	case 0xd5:
		headerSize = 2
		length = 2
	case 0xd6:
		headerSize = 2
		length = 4
	case 0xd7:
		headerSize = 2
		length = 8
	case 0xd8:
		headerSize = 2
		length = 16
	case 0xc7:
		headerSize = 3
		if err := r.fill(headerSize); err != nil {
			return 0, nil, err
		}
		length = int(r.buf[r.pos+1])
	case 0xc8:
		headerSize = 4
		if err := r.fill(headerSize); err != nil {
			return 0, nil, err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
	case 0xc9:
		headerSize = 6
		if err := r.fill(headerSize); err != nil {
			return 0, nil, err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
	default:
		return 0, nil, fmt.Errorf("invalid ext header byte: 0x%02x", b)
	}

	if err := r.fill(headerSize + length); err != nil {
		return 0, nil, err
	}

	var extType int8
	if headerSize == 2 {
		extType = int8(r.buf[r.pos+1])
	} else if headerSize == 3 {
		extType = int8(r.buf[r.pos+2])
	} else if headerSize == 4 {
		extType = int8(r.buf[r.pos+3])
	} else if headerSize == 6 {
		extType = int8(r.buf[r.pos+5])
	}

	data := r.buf[r.pos+headerSize : r.pos+headerSize+length]
	r.consume(headerSize + length)
	return extType, data, nil
}

func (r *Reader) ReadRaw() ([]byte, error) {
	startPos := r.pos

	// Skip over the next item
	err := r.Skip()
	if err != nil {
		return nil, err
	}

	// Return the bytes from startPos to r.pos
	rawBytes := r.buf[startPos:r.pos]
	r.consume(len(rawBytes))

	return rawBytes, nil
}

func (r *Reader) Skip() error {
	if err := r.fill(1); err != nil {
		return err
	}
	b := r.buf[r.pos]

	switch {
	// Nil
	case b == 0xc0:
		r.consume(1)
		return nil
	// Bool
	case b == 0xc2 || b == 0xc3:
		r.consume(1)
		return nil
	// Positive FixInt and Negative FixInt
	case b <= 0x7f || b >= 0xe0:
		r.consume(1)
		return nil
	// Int8, Int16, Int32, Int64
	case b == 0xd0:
		return r.consumeN(2)
	case b == 0xd1:
		return r.consumeN(3)
	case b == 0xd2:
		return r.consumeN(5)
	case b == 0xd3:
		return r.consumeN(9)
	// Uint8, Uint16, Uint32, Uint64
	case b == 0xcc:
		return r.consumeN(2)
	case b == 0xcd:
		return r.consumeN(3)
	case b == 0xce:
		return r.consumeN(5)
	case b == 0xcf:
		return r.consumeN(9)
	// Float32
	case b == 0xca:
		return r.consumeN(5)
	// Float64
	case b == 0xcb:
		return r.consumeN(9)
	// FixStr
	case b >= 0xa0 && b <= 0xbf:
		length := int(b & 0x1f)
		return r.consumeN(1 + length)
	// Str8
	case b == 0xd9:
		if err := r.fill(2); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])
		return r.consumeN(2 + length)
	// Str16
	case b == 0xda:
		if err := r.fill(3); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
		return r.consumeN(3 + length)
	// Str32
	case b == 0xdb:
		if err := r.fill(5); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
		return r.consumeN(5 + length)
	// Bin8
	case b == 0xc4:
		if err := r.fill(2); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])
		return r.consumeN(2 + length)
	// Bin16
	case b == 0xc5:
		if err := r.fill(3); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
		return r.consumeN(3 + length)
	// Bin32
	case b == 0xc6:
		if err := r.fill(5); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
		return r.consumeN(5 + length)
	// FixArray
	case b >= 0x90 && b <= 0x9f:
		length := int(b & 0x0f)
		r.consume(1)
		return r.skipNItems(length)
	// Array16
	case b == 0xdc:
		if err := r.fill(3); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
		r.consume(3)
		return r.skipNItems(length)
	// Array32
	case b == 0xdd:
		if err := r.fill(5); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
		r.consume(5)
		return r.skipNItems(length)
	// FixMap
	case b >= 0x80 && b <= 0x8f:
		length := int(b & 0x0f)
		r.consume(1)
		return r.skipNMapItems(length)
	// Map16
	case b == 0xde:
		if err := r.fill(3); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
		r.consume(3)
		return r.skipNMapItems(length)
	// Map32
	case b == 0xdf:
		if err := r.fill(5); err != nil {
			return err
		}
		length := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
		r.consume(5)
		return r.skipNMapItems(length)
	// FixExt
	case b >= 0xd4 && b <= 0xd8:
		dataSize := 1 << (b - 0xd4)
		return r.consumeN(2 + dataSize)
	// Ext8
	case b == 0xc7:
		if err := r.fill(2); err != nil {
			return err
		}
		dataSize := int(r.buf[r.pos+1])
		return r.consumeN(3 + dataSize)
	// Ext16
	case b == 0xc8:
		if err := r.fill(3); err != nil {
			return err
		}
		dataSize := int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
		return r.consumeN(4 + dataSize)
	// Ext32
	case b == 0xc9:
		if err := r.fill(5); err != nil {
			return err
		}
		dataSize := int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
		return r.consumeN(6 + dataSize)
	default:
		return fmt.Errorf("unsupported type: 0x%02x", b)
	}
}

// consumeN advances the read position by `n` bytes, ensuring enough data is available.
func (r *Reader) consumeN(n int) error {
	if err := r.fill(n); err != nil {
		return err
	}
	r.consume(n)
	return nil
}

// skipNItems skips over `n` MessagePack items.
func (r *Reader) skipNItems(n int) error {
	for i := 0; i < n; i++ {
		if err := r.Skip(); err != nil {
			return err
		}
	}
	return nil
}

// skipNMapItems skips over `n` key-value pairs in a map.
func (r *Reader) skipNMapItems(n int) error {
	for i := 0; i < n; i++ {
		// Skip key
		if err := r.Skip(); err != nil {
			return err
		}
		// Skip value
		if err := r.Skip(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reader) SkipMap() error {
	// Read the map header to determine the number of key-value pairs
	mapLen, err := r.ReadMapHeader()
	if err != nil {
		return fmt.Errorf("SkipMap: failed to read map header: %v", err)
	}

	// Skip all keys and values
	return r.skipNMapItems(mapLen)
}

func (r *Reader) PeekBytes(n int) ([]byte, error) {
	if err := r.fill(n); err != nil {
		return nil, err
	}

	return r.buf[r.pos : r.pos+n], nil
}

var _ io.Reader = (*Reader)(nil)

func (r *Reader) Read(p []byte) (int, error) {
	// If there's no unconsumed data in the buffer, try to fill it.
	if r.pos >= r.size {
		if err := r.fill(1); err != nil {
			// Return EOF if no data remains and it's not an unexpected error
			if err == io.EOF {
				return 0, io.EOF
			}
			return 0, err
		}
	}

	// Determine how much data to copy
	n := copy(p, r.buf[r.pos:r.size])
	r.pos += n // Update the position in the buffer

	return n, nil
}
