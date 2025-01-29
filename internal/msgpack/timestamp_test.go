package msgpack

import (
	"fmt"
	"testing"
	"time"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func sanitizeTimestamp(t time.Time, newOffset int, err error) (time.Time, error) {
	return t.UTC(), err
}

func ExampleReadTimestamp() {
	var buf []byte

	t := time.Date(2025, 01, 01, 1, 2, 3, 0, time.UTC)
	buf = AppendTimestamp(buf, t, Ts32)
	buf = AppendTimestamp(buf, t, Ts64)
	buf = AppendTimestamp(buf, t, Ts96)
	buf = AppendTimestamp(buf, t, TsAuto)
	buf = AppendTimestamp(buf, t, TsFluentd)
	buf = AppendTimestamp(buf, t, TsInt)

	fmt.Println(buf)

	for range 6 {
		fmt.Println(sanitizeTimestamp(ReadTimestamp(buf, 0)))
	}

	// Output:
	//
	// [214 255 103 116 148 11 215 255 25 221 37 2 192 0 0 0 199 12 255 0 0 0 0 0 0 0 0 103 116 148 11 214 255 103 116 148 11 215 0 103 116 148 11 0 0 0 0 206 103 116 148 11]
	// 2025-01-01 01:02:03 +0000 UTC <nil>
	// 2025-01-01 01:02:03 +0000 UTC <nil>
	// 2025-01-01 01:02:03 +0000 UTC <nil>
	// 2025-01-01 01:02:03 +0000 UTC <nil>
	// 2025-01-01 01:02:03 +0000 UTC <nil>
	// 2025-01-01 01:02:03 +0000 UTC <nil>
}

// TestTimestamp exercises encoding/decoding for all TsFormat variants.
func TestTimestamp(t *testing.T) {
	// A range of candidate times for thorough testing.
	testTimes := []time.Time{
		time.Unix(0, 0).UTC(),                       // Unix epoch (UTC)
		time.Unix(1, 0).UTC(),                       // Small positive second
		time.Unix(1e9, 999999999).UTC(),             // Very large second + near 1s nanosecond
		time.Unix(-123456789, 555555).UTC(),         // Negative time
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
		for _, original := range testTimes {
			original := original // pin range variable for subtests

			// Optionally skip negative times for Ts32/TsAuto/TsInt if you want to avoid known overflow.
			if original.Unix() < 0 && (f == Ts32 || f == Ts64) {
				t.Run(f.String()+"_"+original.String(), func(t *testing.T) {
					t.Skipf("Skipping negative epoch test for %v (doesn't support negative)", f)
				})
				continue
			}

			t.Run(f.String()+"_"+original.String(), func(t *testing.T) {
				// Encode the time in the chosen format.
				buf := AppendTimestamp(nil, original, f)

				// Decode the time from the buffer.
				decoded, offset, err := ReadTimestamp(buf, 0)
				if err != nil {
					t.Fatalf("ReadTimestamp error for format=%v, time=%v: %v", f, original, err)
				}

				if offset != len(buf) {
					t.Errorf("Offset mismatch: got %d, want %d (format=%v, time=%v)",
						offset, len(buf), f, original)
				}

				// Convert the decoded time to UTC (to avoid local-time differences).
				decoded = decoded.UTC()

				// Compare the decoded timestamp with the original *at the relevant precision*.
				// Ts32, TsAuto (defaulting to Ts32), and TsInt only store second-level precision.
				if f == Ts32 || f == TsAuto || f == TsInt {
					// Compare only seconds for these formats.
					if decoded.Unix() != original.Unix() {
						t.Errorf("Decoded second mismatch for format=%v.\nWanted: %v\nGot:    %v",
							f, original, decoded)
					}
				} else {
					// Full second + nanosecond comparison for Ts64, Ts96, TsFluentd.
					if decoded.Unix() != original.Unix() || decoded.Nanosecond() != original.Nanosecond() {
						t.Errorf("Decoded time mismatch for format=%v.\nWanted: %v\nGot:    %v",
							f, original, decoded)
					}
				}
			})

			unsafeName := f.String() + "_" + original.String() + "_Unsafe"

			t.Run(unsafeName, func(t *testing.T) {
				// Encode the time in the chosen format.
				buf := AppendTimestamp(nil, original, f)
				_, length, isValueLength := types.Get(buf[0])

				if isValueLength {
					length = 0
				}

				unsafeName = unsafeName

				// Decode the time from the buffer.
				decoded := readTimeUnsafe(buf[0], buf[1+length:])

				// Convert the decoded time to UTC (to avoid local-time differences).
				decoded = decoded.UTC()

				// Compare the decoded timestamp with the original *at the relevant precision*.
				// Ts32, TsAuto (defaulting to Ts32), and TsInt only store second-level precision.
				if f == Ts32 || f == TsAuto || f == TsInt {
					// Compare only seconds for these formats.
					if decoded.Unix() != original.Unix() {
						t.Errorf("Decoded second mismatch for format=%v.\nWanted: %v\nGot:    %v",
							f, original, decoded)
					}
				} else {
					// Full second + nanosecond comparison for Ts64, Ts96, TsFluentd.
					if decoded.Unix() != original.Unix() || decoded.Nanosecond() != original.Nanosecond() {
						t.Errorf("Decoded time mismatch for format=%v.\nWanted: %v\nGot:    %v",
							f, original, decoded)
					}
				}
			})
		}
	}
}
