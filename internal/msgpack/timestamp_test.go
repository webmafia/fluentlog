package msgpack

import (
	"fmt"
	"testing"
	"time"
)

func ExampleReadTimestamp() {
	var buf []byte

	t := time.Date(2025, 01, 01, 1, 2, 3, 0, time.UTC)
	buf = AppendTimestamp(buf, t, TsFluentd)

	fmt.Println(buf)

	dst, _, err := ReadTimestamp(buf, 0)

	if err != nil {
		panic(err)
	}

	fmt.Println(t)
	fmt.Println(dst.UTC())

	// Output: TODO
}

// TestTimestampRoundtrip tests encoding/decoding timestamps across various formats.
func TestTimestamp(t *testing.T) {
	// A range of candidate times for thorough testing.
	testTimes := []time.Time{
		time.Unix(0, 0),                             // Unix epoch
		time.Unix(1, 0),                             // Small positive second
		time.Unix(1e9, 999999999),                   // Very large second + near 1s nanosecond
		time.Unix(-123456789, 555555),               // Negative time
		time.Now().UTC().Truncate(time.Microsecond), // "Current" time truncated for stable ns
	}

	// All TsFormats, including TsAuto, to be tested.
	allFormats := []TsFormat{
		TsAuto,
		Ts32,
		Ts64,
		Ts96,
		TsInt,
		TsFluentd,
	}

	for _, f := range allFormats {
		for _, ttVal := range testTimes {
			ttVal := ttVal // pin the variable for subtest

			t.Run(f.String()+"_"+ttVal.String(), func(t *testing.T) {
				// Encode the time.
				buf := AppendTimestamp(nil, ttVal, f)

				// Decode the time from the buffer.
				decoded, offset, err := ReadTimestamp(buf, 0)
				if err != nil {
					t.Fatalf("ReadTimestamp error for format=%v, time=%v: %v", f, ttVal, err)
				}
				if offset != len(buf) {
					t.Errorf("Offset mismatch: got %d, want %d (format=%v, time=%v)",
						offset, len(buf), f, ttVal)
				}

				// Compare the decoded timestamp with the original
				if decoded.Unix() != ttVal.Unix() || decoded.Nanosecond() != ttVal.Nanosecond() {
					t.Errorf("Decoded time mismatch for format=%v.\nWant: %v\nGot:  %v",
						f, ttVal, decoded)
				}
			})
		}
	}
}
