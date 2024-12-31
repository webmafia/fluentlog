package msgpack

import (
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/webmafia/fast/buffer"
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

// PeekType peeks at the next MessagePack type without consuming any data.
func (r *Reader) PeekType() (t types.Type, err error) {
	c, err := r.b.PeekByte()

	if err != nil {
		return
	}

	t, _, _ = types.Get(c)
	return
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

// ReadArrayHeader reads an array header from the buffered data.
func (r *Reader) ReadArrayHeader() (length int, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	typ, length, err := getLength(c, r.b.ReadBytes)

	if err != nil {
		return
	}

	if typ != types.Array {
		return 0, expectedType(c, types.Array)
	}

	return
}

// ReadMapHeader reads a map header from the buffered data.
func (r *Reader) ReadMapHeader() (length int, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	typ, length, err := getLength(c, r.b.ReadBytes)

	if err != nil {
		return
	}

	if typ != types.Map {
		return 0, expectedType(c, types.Map)
	}

	return
}

// ReadString reads a string from the buffered data.
func (r *Reader) ReadString() (str string, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	typ, length, err := getLength(c, r.b.ReadBytes)

	if err != nil {
		return
	}

	if typ != types.Str {
		return "", expectedType(c, types.Str)
	}

	return r.b.ReadString(length)
}

// ReadInt reads an integer from the buffered data.
func (r *Reader) ReadInt() (i int64, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	// Positive FixInt
	if c <= 0x7f {
		return int64(c), nil
	}

	// Negative FixInt
	if c >= 0xe0 {
		return int64(int8(c)), nil
	}

	typ, length, _ := types.Get(c)

	if typ != types.Int && typ != types.Uint {
		return 0, expectedType(c, types.Int)
	}

	buf, err := r.b.ReadBytes(length)

	if err != nil {
		return
	}

	switch length {

	case 1:
		i = int64(buf[0])

	case 2:
		i = int64(binary.BigEndian.Uint16(buf))

	case 4:
		i = int64(binary.BigEndian.Uint32(buf))

	case 8:
		i = int64(binary.BigEndian.Uint64(buf))
	}

	return
}

// ReadUint reads an unsigned integer from the buffered data.
func (r *Reader) ReadUint() (i uint64, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	// Positive FixInt
	if c <= 0x7f {
		return uint64(c), nil
	}

	typ, length, _ := types.Get(c)

	if typ != types.Uint {
		return 0, expectedType(c, types.Uint)
	}

	buf, err := r.b.ReadBytes(length)

	if err != nil {
		return
	}

	switch length {

	case 1:
		i = uint64(buf[0])

	case 2:
		i = uint64(binary.BigEndian.Uint16(buf))

	case 4:
		i = uint64(binary.BigEndian.Uint32(buf))

	case 8:
		i = binary.BigEndian.Uint64(buf)
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
		return false, expectedType(c, types.Bool)
	}
}

// ReadNil reads/skips a nil value from the buffered data.
func (r *Reader) ReadNil() (err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	if c != 0xc0 {
		return expectedType(c, types.Nil)
	}

	return
}

// ReadBinary reads binary data from the buffered data.
func (r *Reader) ReadBinary() (b []byte, err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	typ, length, err := getLength(c, r.b.ReadBytes)

	if err != nil {
		return
	}

	if typ != types.Bin && typ != types.Str {
		return nil, expectedType(c, types.Bin)
	}

	return r.b.ReadBytes(length)
}

// ReadFloat32 reads a 32-bit floating point number from the buffered data.
func (r *Reader) ReadFloat32() (f float32, err error) {
	buf, err := r.b.ReadBytes(5)

	if err != nil {
		return
	}

	if buf[0] != 0xca {
		return 0, expectedType(buf[0], types.Float)
	}

	return math.Float32frombits(binary.BigEndian.Uint32(buf[1:])), nil
}

// ReadFloat64 reads a 64-bit floating point number from the buffered data.
func (r *Reader) ReadFloat64() (f float64, err error) {
	buf, err := r.b.ReadBytes(9)

	if err != nil {
		return
	}

	if buf[0] != 0xcb {
		return 0, expectedType(buf[0], types.Float)
	}

	return math.Float64frombits(binary.BigEndian.Uint64(buf[1:])), nil
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
			return
		}

		if b[0] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(b[1:5])))
		ns = int64(int32(binary.BigEndian.Uint32(b[5:9])))

	case 0xc7: // ext8
		if b, err = r.b.ReadBytes(10); err != nil {
			return
		}

		if b[0] != 0x08 || b[1] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(b[2:6])))
		ns = int64(int32(binary.BigEndian.Uint32(b[6:10])))

	default:
		if s, err = r.ReadInt(); err != nil {
			return
		}
	}

	return time.Unix(s, ns), nil
}

func (r *Reader) ReadValue() (v Value, err error) {
	start := r.b.Pos()

	if err = r.Skip(); err != nil {
		return
	}

	end := r.b.Pos()

	return Value(r.b.B[start:end]), nil
}

func (r *Reader) Skip() (err error) {
	c, err := r.b.ReadByte()

	if err != nil {
		return
	}

	typ, length, err := getLength(c, r.b.ReadBytes)

	if err != nil {
		return
	}

	if typ == types.Array {
		for range length {
			if err = r.Skip(); err != nil {
				return
			}
		}
	} else if typ == types.Map {
		for range length * 2 {
			if err = r.Skip(); err != nil {
				return
			}
		}
	} else {
		_, err = r.b.ReadBytes(length)
	}

	return
}
