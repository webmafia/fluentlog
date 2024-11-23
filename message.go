package fluentlog

import (
	"encoding/binary"
	"iter"
	"time"
)

type Message struct {
	buf []byte
}

func NewMessage() Message {
	msg := Message{
		buf: make([]byte, 11, 1024),
	}

	msg.buf[0] = 0x92
	msg.buf[1] = 0xcf

	msg.setTimestamp()
	msg.buf = append(msg.buf, 0x80) // Initial empty map (fixmap with 0 fields)
	return msg
}

func (msg *Message) Reset() {
	msg.buf = msg.buf[:11]          // Reset to keep array header and timestamp only
	msg.buf = append(msg.buf, 0x80) // Reset to empty map (fixmap with 0 fields)
}

func (msg *Message) write(buf ...byte) {
	msg.buf = append(msg.buf, buf...)
}

func (msg *Message) setTimestamp() {
	binary.BigEndian.PutUint64(msg.buf[2:], uint64(time.Now().Unix()))
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

// incNumFields increments the field count and updates the map header at position 11
func (msg *Message) incNumFields() {
	numFields := msg.NumFields() + 1
	switch {
	case numFields <= 15:
		// Update fixmap header (0x80–0x8f)
		msg.buf[11] = 0x80 | byte(numFields)
	case numFields <= 0xffff:
		// Transition to map16
		if msg.buf[11] < 0xde {
			// Shift data to make space for the larger header
			msg.buf = append(msg.buf[:11], append([]byte{0xde, 0x00, 0x00}, msg.buf[12:]...)...)
		}
		// Write the updated number of fields (map16, 2 bytes)
		binary.BigEndian.PutUint16(msg.buf[12:14], uint16(numFields))
	case numFields <= 0xffffffff:
		// Transition to map32
		if msg.buf[11] < 0xdf {
			// Shift data to make space for the larger header
			msg.buf = append(msg.buf[:11], append([]byte{0xdf, 0x00, 0x00, 0x00, 0x00}, msg.buf[12:]...)...)
		}
		// Write the updated number of fields (map32, 4 bytes)
		binary.BigEndian.PutUint32(msg.buf[12:16], uint32(numFields))
	}
}

// AddField adds a key-value pair to the map in the MsgPack message
func (msg *Message) AddField(key, value string) {
	msg.writeString(key)
	msg.writeString(value)
	msg.incNumFields()
}

// Helper function to write a MsgPack string
func (msg *Message) writeString(s string) {
	strLen := len(s)
	if strLen <= 31 {
		msg.write(0xa0 | byte(strLen))
	} else if strLen <= 255 {
		msg.write(0xd9, byte(strLen))
	} else {
		msg.write(0xda, byte(strLen>>8), byte(strLen))
	}
	msg.write([]byte(s)...)
}

func (msg *Message) Fields() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		// Start reading from the map data in msg.buf, assuming the map begins at index 11.
		pos := 12                    // Starting position after the map header at index 11
		numFields := msg.NumFields() // Get the number of key-value pairs

		for i := 0; i < numFields; i++ {
			// Decode the key as a string
			key, n := decodeString(msg.buf[pos:])
			pos += n

			// Decode the value as a string
			value, n := decodeString(msg.buf[pos:])
			pos += n

			// Yield the key-value pair to the callback
			if !yield(key, value) {
				break
			}
		}
	}
}

// Helper function to decode a MsgPack-encoded string
func decodeString(buf []byte) (string, int) {
	// Check the first byte to determine string length format
	if len(buf) == 0 {
		return "", 0
	}

	switch {
	case buf[0]>>5 == 0b101: // fixstr (0xa0 to 0xbf)
		strLen := int(buf[0] & 0x1f)
		return string(buf[1 : 1+strLen]), 1 + strLen
	case buf[0] == 0xd9: // str8
		strLen := int(buf[1])
		return string(buf[2 : 2+strLen]), 2 + strLen
	case buf[0] == 0xda: // str16
		strLen := int(binary.BigEndian.Uint16(buf[1:3]))
		return string(buf[3 : 3+strLen]), 3 + strLen
	case buf[0] == 0xdb: // str32
		strLen := int(binary.BigEndian.Uint32(buf[1:5]))
		return string(buf[5 : 5+strLen]), 5 + strLen
	default:
		return "", 0 // Unsupported type
	}
}
