package fluentlog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"time"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Message struct {
	buf []byte
}

func NewMessage(tag string) Message {
	msg := Message{
		buf: make([]byte, 0, 1024),
	}

	// Append array header for an array of 3 elements: [tag, timestamp, record]
	msg.buf = msgpack.AppendArray(msg.buf, 3)

	// Append tag
	msg.buf = msgpack.AppendString(msg.buf, tag)

	// Append timestamp
	msg.buf = msgpack.AppendTimestamp(msg.buf, time.Now())

	// Append initial map header (fixmap with zero elements)
	msg.buf = append(msg.buf, 0x80) // fixmap with zero elements

	return msg
}

func (msg *Message) Reset(tag string) {
	msg.buf = msg.buf[:0]
	// Reinitialize the message as in NewMessage
	msg.buf = msgpack.AppendArray(msg.buf, 3)
	msg.buf = msgpack.AppendString(msg.buf, tag)
	msg.buf = msgpack.AppendTimestamp(msg.buf, time.Now())
	// Append initial map header (fixmap with zero elements)
	msg.buf = append(msg.buf, 0x80) // fixmap with zero elements
}

// NumFields reads the number of fields in the map directly from the buffer at position 11
func (msg *Message) NumFields() int {
	switch msg.buf[11] {
	case 0x80: // fixmap with 0 fields
		return 0
	case 0x8f: // fixmap with 15 fields
		return 15
	case 0xde: // map16, up to (2^16 - 1) fields
		return int(binary.BigEndian.Uint16(msg.buf[12:14]))
	case 0xdf: // map32, up to (2^32 - 1) fields
		return int(binary.BigEndian.Uint32(msg.buf[12:16]))
	default:
		return int(msg.buf[11] & 0x0f) // fixmap with fields 1–14
	}
}

func (msg *Message) AddField(key string, value any) error {
	// Find the map header position and the current number of fields
	mapHeaderPos, numFields, err := msg.findMapHeader()
	if err != nil {
		return err
	}

	// Append the key and value to the buffer
	msg.buf = msgpack.AppendString(msg.buf, key)

	// Append the value based on its type
	switch v := value.(type) {
	case string:
		msg.buf = msgpack.AppendString(msg.buf, v)
	case int, int8, int16, int32, int64:
		msg.buf = msgpack.AppendInt(msg.buf, reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		msg.buf = msgpack.AppendUint(msg.buf, reflect.ValueOf(v).Uint())
	case float32:
		msg.buf = msgpack.AppendFloat32(msg.buf, v)
	case float64:
		msg.buf = msgpack.AppendFloat64(msg.buf, v)
	case bool:
		msg.buf = msgpack.AppendBool(msg.buf, v)
	case nil:
		msg.buf = msgpack.AppendNil(msg.buf)
	case time.Time:
		msg.buf = msgpack.AppendTimestamp(msg.buf, v)
	default:
		return errors.New("unsupported value type")
	}

	// Increment the number of fields
	numFields++

	// Update the map header at mapHeaderPos
	err = msg.updateMapHeader(mapHeaderPos, numFields)
	if err != nil {
		return err
	}

	return nil
}

func (msg *Message) updateMapHeader(mapHeaderPos int, numFields int) error {
	// Determine the new map header
	var newHeader []byte
	var newHeaderSize int

	switch {
	case numFields <= 15:
		// fixmap
		newHeader = []byte{0x80 | byte(numFields)}
		newHeaderSize = 1
	case numFields <= 0xffff:
		// map16
		newHeader = []byte{0xde, byte(numFields >> 8), byte(numFields)}
		newHeaderSize = 3
	case numFields <= 0xffffffff:
		// map32
		newHeader = []byte{
			0xdf,
			byte(numFields >> 24),
			byte(numFields >> 16),
			byte(numFields >> 8),
			byte(numFields),
		}
		newHeaderSize = 5
	default:
		return errors.New("too many fields in map")
	}

	// Read the existing map header to determine its size
	b := msg.buf[mapHeaderPos]
	var oldHeaderSize int
	switch {
	case b >= 0x80 && b <= 0x8f:
		oldHeaderSize = 1
	case b == 0xde:
		oldHeaderSize = 3
	case b == 0xdf:
		oldHeaderSize = 5
	default:
		return fmt.Errorf("invalid map header at position %d", mapHeaderPos)
	}

	sizeDiff := newHeaderSize - oldHeaderSize
	if sizeDiff > 0 {
		// Need to expand the buffer
		msg.buf = append(msg.buf, make([]byte, sizeDiff)...) // Increase buffer size
		// Shift the data forward
		copy(msg.buf[mapHeaderPos+newHeaderSize:], msg.buf[mapHeaderPos+oldHeaderSize:len(msg.buf)-sizeDiff])
	} else if sizeDiff < 0 {
		// Need to shrink the buffer
		copy(msg.buf[mapHeaderPos+newHeaderSize:], msg.buf[mapHeaderPos+oldHeaderSize:])
		msg.buf = msg.buf[:len(msg.buf)+sizeDiff] // Reduce buffer size
	}
	// Else sizeDiff == 0, no need to adjust the buffer

	// Replace the map header
	copy(msg.buf[mapHeaderPos:], newHeader)

	return nil
}

func (msg *Message) Fields() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		var err error
		var offset int

		// Read the array header
		if _, offset, err = msgpack.ReadArrayHeader(msg.buf, offset); err != nil {
			return
		}

		// Skip the tag
		if _, offset, err = msgpack.ReadString(msg.buf, offset); err != nil {
			return
		}

		// Skip the timestamp
		if _, offset, err = msgpack.ReadTimestamp(msg.buf, offset); err != nil {
			return
		}
		// Read the map header
		numFields, offset, err := msgpack.ReadMapHeader(msg.buf, offset)
		if err != nil {
			return
		}
		for i := 0; i < numFields; i++ {
			// Read key
			var key string

			if key, offset, err = msgpack.ReadString(msg.buf, offset); err != nil {
				return
			}
			// Read value
			if offset >= len(msg.buf) {
				return
			}
			b := msg.buf[offset]
			var value any
			switch {
			case b >= 0xa0 && b <= 0xbf, b == 0xd9, b == 0xda, b == 0xdb:
				// String
				value, offset, err = msgpack.ReadString(msg.buf, offset)
			case b <= 0x7f, b >= 0xe0, b == 0xcc, b == 0xcd, b == 0xce, b == 0xcf, b == 0xd0, b == 0xd1, b == 0xd2, b == 0xd3:
				// Integer
				value, offset, err = msgpack.ReadInt(msg.buf, offset)
			case b == 0xca:
				// Float32
				value, offset, err = msgpack.ReadFloat32(msg.buf, offset)
			case b == 0xcb:
				// Float64
				value, offset, err = msgpack.ReadFloat64(msg.buf, offset)
			case b == 0xc2, b == 0xc3:
				// Boolean
				value, offset, err = msgpack.ReadBool(msg.buf, offset)
			case b == 0xc0:
				// Nil
				value = nil
				offset, err = msgpack.ReadNil(msg.buf, offset)
			case b == 0xd6, b == 0xd7, b == 0xd8:
				// Extension types (could be timestamp)
				typ, data, newOffset, err := msgpack.ReadExt(msg.buf, offset)
				if err != nil {
					return
				}
				offset = newOffset
				if typ == -1 {
					// Timestamp
					if len(data) == 4 {
						sec := int64(binary.BigEndian.Uint32(data))
						value = time.Unix(sec, 0).UTC()
					} else if len(data) == 8 {
						sec := int64(binary.BigEndian.Uint64(data))
						value = time.Unix(sec, 0).UTC()
					} else {
						// Handle other timestamp formats if needed
						value = data
					}
				} else {
					// Other extension types
					value = data
				}
			default:
				// Unsupported type
				return
			}
			if err != nil {
				return
			}
			if !yield(key, value) {
				break
			}
		}
	}
}

func (msg *Message) findMapHeader() (mapHeaderPos int, numFields int, err error) {
	offset := 0
	// Read the array header
	_, offset, err = msgpack.ReadArrayHeader(msg.buf, offset)
	if err != nil {
		return
	}

	// Skip the tag
	_, offset, err = msgpack.ReadString(msg.buf, offset)
	if err != nil {
		return
	}

	// Skip the timestamp
	_, offset, err = msgpack.ReadTimestamp(msg.buf, offset)
	if err != nil {
		return
	}

	// Now, offset is at the position of the map header
	mapHeaderPos = offset

	// Read the map header
	numFields, _, err = msgpack.ReadMapHeader(msg.buf, offset)
	if err != nil {
		return
	}

	return mapHeaderPos, numFields, nil
}
