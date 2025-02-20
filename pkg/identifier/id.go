package identifier

import (
	"time"
)

// 32 bits unix seconds + 32 bits sequence (where first 10 bits are milliseconds)
type ID uint64

// newID generates a new ID from the given timestamp and sequence number
func newID(ts time.Time, seq uint32) ID {
	// Combine the timestamp, milliseconds, and sequence into a 64-bit ID
	return ID((uint64(ts.Unix()) << 32) | (uint64(ts.Nanosecond()/1_000_000) << 22) | (uint64(seq) & 0x3FFFFF))
}

// Unix timestamp in seconds precision
func (id ID) Unix() uint32 {
	return uint32(id >> 32)
}

// Extracts sequence number (last 22 bits)
func (id ID) Seq() uint32 {
	return uint32(id & 0x3FFFFF)
}

// Converts ID to time.Time
func (id ID) Time() time.Time {
	return time.Unix(int64(id>>32), int64((id>>22)&0x3FF)*1_000_000) // Convert milliseconds to nanoseconds
}

func (id ID) Uint64() uint64 {
	return uint64(id)
}
