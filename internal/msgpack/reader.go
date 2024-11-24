package msgpack

import (
	"errors"
	"io"
	"time"
)

type Reader struct {
	r    io.Reader
	buf  []byte
	size int // Number of valid bytes in `buf`
	pos  int // Current read position in `buf`
}

// NewReader initializes a new Reader with the provided io.Reader and buffer size.
func NewReader(r io.Reader, bufferSize int) *Reader {
	return &Reader{
		r:   r,
		buf: make([]byte, bufferSize),
	}
}

// fill ensures that the buffer contains at least `n` bytes starting from the current position.
func (r *Reader) fill(n int) error {
	remaining := r.size - r.pos
	if remaining >= n {
		return nil // Already have enough data
	}

	read, err := io.ReadAtLeast(r.r, r.buf[r.size:], remaining)

	if err != nil {
		return err
	}

	r.size += read

	return nil
}

// consume advances the position by `n` bytes without modifying the buffer.
func (r *Reader) consume(n int) {
	r.pos += n
}

// Release resets the buffer and position after the caller confirms that slices are no longer needed.
func (r *Reader) Release() {
	copy(r.buf, r.buf[r.pos:r.size]) // Preserve unconsumed data
	r.size -= r.pos
	r.pos = 0
}

// readLength determines the length of a variable-length MessagePack object.
func (r *Reader) readLength() (length int, headerSize int, err error) {
	if err := r.fill(1); err != nil {
		return 0, 0, err
	}

	b := r.buf[r.pos]
	switch {
	case b >= 0xa0 && b <= 0xbf: // fixstr
		length = int(b & 0x1f)
		headerSize = 1
	case b == 0xd9: // str8
		if err := r.fill(2); err != nil {
			return 0, 0, err
		}
		length = int(r.buf[r.pos+1])
		headerSize = 2
	case b == 0xda: // str16
		if err := r.fill(3); err != nil {
			return 0, 0, err
		}
		length = int(r.buf[r.pos+1])<<8 | int(r.buf[r.pos+2])
		headerSize = 3
	case b == 0xdb: // str32
		if err := r.fill(5); err != nil {
			return 0, 0, err
		}
		length = int(r.buf[r.pos+1])<<24 | int(r.buf[r.pos+2])<<16 | int(r.buf[r.pos+3])<<8 | int(r.buf[r.pos+4])
		headerSize = 5
	default:
		return 0, 0, errors.New("unsupported or invalid type for length determination")
	}
	return length, headerSize, nil
}

// ReadArrayHeader reads an array header from the buffered data.
func (r *Reader) ReadArrayHeader() (int, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	length, newOffset, err := ReadArrayHeader(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, err
	}

	r.consume(newOffset)
	return length, nil
}

// ReadMapHeader reads a map header from the buffered data.
func (r *Reader) ReadMapHeader() (int, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	length, newOffset, err := ReadMapHeader(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, err
	}

	r.consume(newOffset)
	return length, nil
}

// ReadString reads a string from the buffered data.
func (r *Reader) ReadString() (string, error) {
	length, headerSize, err := r.readLength()
	if err != nil {
		return "", err
	}

	// Total bytes required include the header size and the length of the string data
	totalBytesNeeded := headerSize + length
	if err := r.fill(totalBytesNeeded); err != nil {
		return "", err
	}

	s, newOffset, err := ReadString(r.buf[r.pos:r.pos+totalBytesNeeded], 0)
	if err != nil {
		return "", err
	}

	r.consume(newOffset)
	return s, nil
}

// ReadInt reads an integer from the buffered data.
func (r *Reader) ReadInt() (int64, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	value, newOffset, err := ReadInt(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, err
	}

	r.consume(newOffset)
	return value, nil
}

// ReadUint reads an unsigned integer from the buffered data.
func (r *Reader) ReadUint() (uint64, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	value, newOffset, err := ReadUint(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, err
	}

	r.consume(newOffset)
	return value, nil
}

// ReadBool reads a boolean value from the buffered data.
func (r *Reader) ReadBool() (bool, error) {
	if err := r.fill(1); err != nil {
		return false, err
	}

	value, newOffset, err := ReadBool(r.buf[r.pos:r.size], 0)
	if err != nil {
		return false, err
	}

	r.consume(newOffset)
	return value, nil
}

// ReadNil reads a nil value from the buffered data.
func (r *Reader) ReadNil() error {
	if err := r.fill(1); err != nil {
		return err
	}

	newOffset, err := ReadNil(r.buf[r.pos:r.size], 0)
	if err != nil {
		return err
	}

	r.consume(newOffset)
	return nil
}

// ReadBinary reads binary data from the buffered data.
func (r *Reader) ReadBinary() ([]byte, error) {
	if err := r.fill(1); err != nil {
		return nil, err
	}

	data, newOffset, err := ReadBinary(r.buf[r.pos:r.size], 0)
	if err != nil {
		return nil, err
	}

	r.consume(newOffset)
	return data, nil
}

// ReadTimestamp reads a timestamp value from the buffered data.
func (r *Reader) ReadTimestamp() (time.Time, error) {
	if err := r.fill(1); err != nil {
		return time.Time{}, err
	}

	t, newOffset, err := ReadTimestamp(r.buf[r.pos:r.size], 0)
	if err != nil {
		return time.Time{}, err
	}

	r.consume(newOffset)
	return t, nil
}

// ReadExt reads an extension object from the buffered data.
func (r *Reader) ReadExt() (int8, []byte, error) {
	if err := r.fill(1); err != nil {
		return 0, nil, err
	}

	typ, data, newOffset, err := ReadExt(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, nil, err
	}

	r.consume(newOffset)
	return typ, data, nil
}

// ReadFloat32 reads a 32-bit floating point number from the buffered data.
func (r *Reader) ReadFloat32() (float32, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	value, newOffset, err := ReadFloat32(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, err
	}

	r.consume(newOffset)
	return value, nil
}

// ReadFloat64 reads a 64-bit floating point number from the buffered data.
func (r *Reader) ReadFloat64() (float64, error) {
	if err := r.fill(1); err != nil {
		return 0, err
	}

	value, newOffset, err := ReadFloat64(r.buf[r.pos:r.size], 0)
	if err != nil {
		return 0, err
	}

	r.consume(newOffset)
	return value, nil
}
