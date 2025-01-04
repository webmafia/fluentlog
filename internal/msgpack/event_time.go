package msgpack

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

// Append a Fluentd Forward EventTime with millisecond precision. This isn't really part of
// the MessagePack specification, but an Extension type. Located here for convenience.
func AppendEventTime(dst []byte, t time.Time) []byte {
	s, ns := uint32(t.Unix()), uint32(t.Nanosecond())

	return append(dst,

		// Append the fixext8 header and type
		0xd7,
		0x00,

		// Append the seconds as a 32-bit big-endian integer
		byte(s>>24),
		byte(s>>16),
		byte(s>>8),
		byte(s),

		// Append the nanoseconds as a 32-bit big-endian integer
		byte(ns>>24),
		byte(ns>>16),
		byte(ns>>8),
		byte(ns),
	)
}

func AppendEventTimeShort(dst []byte, t time.Time) []byte {
	return AppendInt(dst, t.UTC().Unix())
}

// ReadTimestamp reads a timestamp value from the given byte slice starting at the specified offset.
// It supports both EventTime (ext type 0) and integer timestamps.
func ReadEventTime(src []byte, offset int) (t time.Time, newOffset int, err error) {
	if offset >= len(src) {
		err = io.ErrUnexpectedEOF
		return
	}

	b := src[offset]
	var s, ns int64

	switch b {
	case 0xd7: // fixext8
		if offset+10 > len(src) {
			err = io.ErrUnexpectedEOF
			return
		}

		if src[offset+1] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(src[offset+2 : offset+6])))
		ns = int64(int32(binary.BigEndian.Uint32(src[offset+6 : offset+10])))
		newOffset = offset + 10

	case 0xc7: // ext8
		if offset+11 > len(src) {
			err = io.ErrUnexpectedEOF
			return
		}

		if src[offset+1] != 0x08 || src[offset+2] != 0x00 {
			err = errors.New("invalid timestamp type")
			return
		}

		s = int64(int32(binary.BigEndian.Uint32(src[offset+3 : offset+7])))
		ns = int64(int32(binary.BigEndian.Uint32(src[offset+7 : offset+11])))
		newOffset = offset + 11

	default:
		var intVal int64
		intVal, newOffset, err = ReadInt(src, offset)
		if err != nil {
			return
		}
		s = intVal
	}

	t = time.Unix(s, ns)
	return
}
