package msgpack

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type TsFormat uint8

const (
	TsAuto             TsFormat = iota
	Ts32                        // 0xd6: Unix timestamp with seconds precision
	Ts64                        // 0xd7: Unix timestamp with nanoseconds precision (compact)
	Ts96                        // 0xc7: Unix timestamp with nanoseconds precision (full)
	TsInt                       // Unix timestamp with seconds precision as regular integer
	TsForwardEventTime          // Fluntd Forward EventTime
)

const msgpackTimestamp = 0xff
const forwardEventTime = 0x00

// AppendEventTime appends a Fluentd Forward EventTime (with seconds and nanoseconds) to `dst`.
// Returns the updated byte slice.
func AppendTimestamp(dst []byte, t time.Time, format ...TsFormat) []byte {
	var f TsFormat

	if len(format) > 0 {
		f = format[0]
	}

	switch f {

	case TsAuto:
		f = Ts32
		fallthrough

	case Ts32:
		s := uint32(t.Unix())

		dst = append(dst,

			// Append the header and type
			0xd6,
			msgpackTimestamp,

			// Append the seconds as a 32-bit big-endian unsigned integer
			byte(s>>24),
			byte(s>>16),
			byte(s>>8),
			byte(s),
		)

	case Ts64:
		// Encode nanoseconds into the lower 30 bits
		ns := uint64(t.Nanosecond() & 0x3FFFFFFF)

		// Encode seconds into the upper 34 bits
		s := uint64(t.Unix()&0x3FFFFFFFF) << 30

		// Combine nanoseconds and seconds
		v := ns | s

		dst = append(dst,

			// Append the header and type
			0xd7,
			msgpackTimestamp,

			// Append the combination as a 64-bit big-endian unsigned integer
			byte(v>>56),
			byte(v>>48),
			byte(v>>40),
			byte(v>>32),
			byte(v>>24),
			byte(v>>16),
			byte(v>>8),
			byte(v),
		)

	case Ts96:
		s, ns := t.Unix(), uint32(t.Nanosecond())

		dst = append(dst,

			// Append the header, length and type
			0xc7,
			12,
			msgpackTimestamp,

			// Append the nanoseconds as a 32-bit big-endian unsigned integer
			byte(ns>>24),
			byte(ns>>16),
			byte(ns>>8),
			byte(ns),

			// Append the combination as a 64-bit big-endian signed integer
			byte(s>>56),
			byte(s>>48),
			byte(s>>40),
			byte(s>>32),
			byte(s>>24),
			byte(s>>16),
			byte(s>>8),
			byte(s),
		)

	case TsInt:
		dst = AppendInt(dst, t.Unix())

	case TsForwardEventTime:
		s, ns := uint32(t.Unix()), uint32(t.Nanosecond())

		dst = append(dst,

			// Append the header and type
			0xd7,
			forwardEventTime,

			// Append the seconds as a 32-bit big-endian unsigned integer
			byte(s>>24),
			byte(s>>16),
			byte(s>>8),
			byte(s),

			// Append the nanoseconds as a 32-bit big-endian unsigned integer
			byte(ns>>24),
			byte(ns>>16),
			byte(ns>>8),
			byte(ns),
		)
	}

	return dst
}

// ReadEventTime reads a timestamp from `src` starting at `offset`.
// Supports both Fluentd Forward EventTime and standard Unix timestamps.
// Returns the decoded time, the new offset, and an error if the data is invalid or incomplete.
func ReadTimestamp(src []byte, offset int) (t time.Time, newOffset int, err error) {
	if offset >= len(src) {
		err = io.ErrUnexpectedEOF
		return
	}

	origOffset := offset
	c := src[offset]
	_, length, isValueLength := types.Get(c)

	offset++

	if !isValueLength {
		if offset+length > len(src) {
			err = ErrShortBuffer
			return
		}

		l := length
		length = int(uintFromBuf[uint](src[offset : offset+l]))
		offset += l
	}

	if offset+length > len(src) {
		err = ErrShortBuffer
		return
	}

	var s, ns int64

	switch c {

	case 0xd6: // Ts32
		if h := src[offset]; h != msgpackTimestamp {
			err = expectedExtType(h, msgpackTimestamp)
			return
		}

		offset++

		s = int64(binary.BigEndian.Uint32(src[offset : offset+4]))
		offset += 4

	case 0xd7: // Ts64 or Forward EventTime
		h := src[offset]
		offset++

		if h == msgpackTimestamp {
			// Read the combined 64-bit value
			combined := binary.BigEndian.Uint64(src[offset:])

			// Extract nanoseconds (lower 30 bits)
			ns = int64(combined & 0x3FFFFFFF)

			// Extract seconds (upper 34 bits)
			s = int64(combined >> 30)

		} else if h == forwardEventTime {
			s = int64(int32(binary.BigEndian.Uint32(src[offset : offset+4])))
			ns = int64(int32(binary.BigEndian.Uint32(src[offset+4 : offset+8])))

		} else {
			err = expectedExtType(h, msgpackTimestamp)
			return
		}

		offset += 8

	case 0xc7: // ext8
		h := src[offset]
		offset++

		if h == msgpackTimestamp {
			ns = int64(binary.BigEndian.Uint32(src[offset : offset+4]))
			s = int64(binary.BigEndian.Uint64(src[offset+4 : offset+12]))

		} else if h == forwardEventTime {
			s = int64(int32(binary.BigEndian.Uint32(src[offset : offset+4])))
			ns = int64(int32(binary.BigEndian.Uint32(src[offset+4 : offset+8])))

		} else {
			err = expectedExtType(h, msgpackTimestamp)
			return
		}

		offset += length

	default:
		if s, newOffset, err = ReadInt(src, origOffset); err != nil {
			return
		}
	}

	t = time.Unix(s, ns)
	return
}
