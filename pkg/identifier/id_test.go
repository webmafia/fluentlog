package identifier

import (
	"testing"
	"time"
)

// TestIDEncodingDecoding verifies that encoding an ID with newID()
// and then extracting its components gives back the original values.
func TestIDEncodingDecoding(t *testing.T) {
	// Define test cases with various timestamps and sequence numbers
	testCases := []struct {
		timestamp time.Time
		seq       uint32
	}{
		{time.Unix(1708400000, 123_000_000), 42},      // Normal case
		{time.Unix(1600000000, 999_000_000), 1023},    // Edge case: high milliseconds
		{time.Unix(1500000000, 0), 0},                 // Edge case: zero milliseconds, zero sequence
		{time.Unix(1800000000, 500_000_000), 4194303}, // Max sequence (22 bits)
	}

	for _, tc := range testCases {
		// Generate ID
		id := newID(tc.timestamp, tc.seq)

		// Validate Unix timestamp
		if id.Unix() != uint32(tc.timestamp.Unix()) {
			t.Errorf("Unix() mismatch: got %d, want %d", id.Unix(), uint32(tc.timestamp.Unix()))
		}

		// Validate sequence number
		if id.Seq() != (tc.seq & 0x3FFFFF) { // Ensure it fits 22-bit mask
			t.Errorf("Seq() mismatch: got %d, want %d", id.Seq(), tc.seq&0x3FFFFF)
		}

		// Validate reconstructed time
		reconstructedTime := id.Time()
		if reconstructedTime.Unix() != tc.timestamp.Unix() {
			t.Errorf("Time() mismatch: Unix seconds got %d, want %d", reconstructedTime.Unix(), tc.timestamp.Unix())
		}

		// Validate milliseconds part
		expectedMS := tc.timestamp.Nanosecond() / 1_000_000
		reconstructedMS := reconstructedTime.Nanosecond() / 1_000_000
		if reconstructedMS != expectedMS {
			t.Errorf("Time() mismatch: milliseconds got %d, want %d", reconstructedMS, expectedMS)
		}
	}
}
