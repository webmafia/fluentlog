package forward

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/valyala/bytebufferpool"
	"github.com/webmafia/fluentlog/internal"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

var _ io.WriterTo = Message{}

type Message struct {
	buf *bytebufferpool.ByteBuffer
}

func NewMessage(tag string, ts time.Time) Message {
	msg := Message{
		buf: make([]byte, 0, 1024),
	}

	msg.Reset(tag, ts)

	return msg
}

func (msg *Message) Reset(tag string, ts time.Time) {
	msg.buf = msg.buf[:0]
	msg.buf = msgpack.AppendArray(msg.buf, 3)
	msg.buf = msgpack.AppendString(msg.buf, tag)
	msg.buf = msgpack.AppendTimestamp(msg.buf, ts)
	msg.buf = msgpack.AppendMap(msg.buf, 0)
}

func (msg *Message) Data() (tag, ts, fields Value) {
	_, off, _ := msgpack.ReadArrayHeader(msg.buf, 0)

	l, _ := msgpack.GetMsgpackValueLength(msg.buf[off:])
	tag = msg.buf[off : off+l]
	off += l

	l, _ = msgpack.GetMsgpackValueLength(msg.buf[off:])
	ts = msg.buf[off : off+l]
	off += l

	l, _ = msgpack.GetMsgpackValueLength(msg.buf[off:])
	fields = msg.buf[off : off+l]

	return
}

func (msg *Message) Tag() string {
	tag, _, _ := msg.Data()
	return tag.Str()
}

func (msg *Message) Time() time.Time {
	_, ts, _ := msg.Data()
	return ts.Time()
}

func (msg *Message) Fields() Value {
	_, _, fields := msg.Data()
	return fields
}

func (msg *Message) NumFields() int {
	return msg.Fields().Len()
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
	case []byte:
		msg.buf = msgpack.AppendBinary(msg.buf, v)
	case internal.TextAppender:
		msg.buf = msgpack.AppendTextAppender(msg.buf, v)
	case encoding.TextMarshaler:
		val, _ := v.MarshalText()
		msg.buf = msgpack.AppendString(msg.buf, internal.B2S(val))
	case fmt.Stringer:
		msg.buf = msgpack.AppendString(msg.buf, v.String())
	case internal.BinaryAppender:
		msg.buf = msgpack.AppendBinaryAppender(msg.buf, v)
	case encoding.BinaryMarshaler:
		val, _ := v.MarshalBinary()
		msg.buf = msgpack.AppendBinary(msg.buf, val)
	default:
		msg.buf = msgpack.AppendUnknownString(msg.buf, func(dst []byte) []byte {
			return fmt.Appendf(dst, "%v", v)
		})
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
	_, offset, err = msgpack.ReadInt(msg.buf, offset)
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

func (msg Message) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(msg.buf)
	return int64(n), err
}
