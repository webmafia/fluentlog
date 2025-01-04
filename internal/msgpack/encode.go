package msgpack

import (
	"time"
)

func AppendTimestamp(dst []byte, t time.Time) []byte {
	return AppendInt(dst, t.UTC().Unix())
}

func AppendExtendedTimestamp(dst []byte, t time.Time) []byte {
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
