package identifier

import (
	"testing"
	"time"
)

func FuzzID(f *testing.F) {
	// Seed with some initial values.
	f.Add(int64(1600000000), int64(123456789), uint32(42))
	f.Add(int64(0), int64(0), uint32(0))
	f.Add(time.Now().Unix(), int64(987654321), uint32(1<<22-1)) // max seq value for 22 bits

	f.Fuzz(func(t *testing.T, sec int64, nsec int64, seq uint32) {
		if sec < 0 || nsec < 0 {
			t.Skip()
		}

		// Clamp nanoseconds to [0, 1e9)
		nsec = nsec % 1_000_000_000

		// Create a timestamp and generate an ID.
		ts := time.Unix(sec, nsec)
		id := newID(ts, seq)

		// Verify that the Unix seconds are preserved.
		if got, want := id.Unix(), uint32(ts.Unix()); got != want {
			t.Errorf("Unix mismatch: got %d, want %d", got, want)
		}

		// Verify that the Time method returns a timestamp truncated to milliseconds.
		ms := ts.Nanosecond() / 1_000_000
		expectedTime := time.Unix(ts.Unix(), int64(ms)*1_000_000)
		if got, want := id.Time(), expectedTime; !got.Equal(want) {
			t.Errorf("Time mismatch: got %v, want %v", got, want)
		}

		// Verify that the sequence number is limited to 22 bits.
		if got, want := id.Seq(), seq&0x3FFFFF; got != want {
			t.Errorf("Seq mismatch: got %d, want %d", got, want)
		}

		// Verify that AppendText and DecodeText form a round-trip.
		decoded, err := IDFromString(id.String())
		if err != nil {
			t.Fatalf("DecodeText error: %v", err)
		}
		if decoded != id {
			t.Errorf("Decoded ID mismatch: got %v, want %v", decoded, id)
		}
	})
}
