package msgpack

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

func TestAppendFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected []byte
	}{
		{"Float32 Example", 3.14159, []byte{0xca, 0x40, 0x49, 0x0f, 0xdb}},
		{"Float64 Example", 1.7976931348623157e+308, []byte{0xcb, 0x7f, 0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendFloat(nil, tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadFloat(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedValue  float64
		expectedOffset int
		expectedErr    error
	}{
		{"Valid Float32", []byte{0xca, 0x40, 0x49, 0x0f, 0xdb}, 0, 3.14159, 5, nil},
		{"Valid Float64", []byte{0xcb, 0x7f, 0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 0, 1.7976931348623157e+308, 9, nil},
		{"Invalid Header", []byte{0xcc, 0x00}, 0, 0, 0, ErrInvalidHeaderByte},
		{"Short Float32 Buffer", []byte{0xca, 0x40, 0x49}, 0, 0, 0, ErrShortBuffer},
		{"Short Float64 Buffer", []byte{0xcb, 0x7f, 0xef, 0xff, 0xff}, 0, 0, 0, ErrShortBuffer},
		{"Empty Input", []byte{}, 0, 0, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, newOffset, err := ReadFloat(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify decoded value and offset
			if !floatEqual(value, tt.expectedValue) {
				t.Errorf("expected value %f, got %f", tt.expectedValue, value)
			}
			if newOffset != tt.expectedOffset {
				t.Errorf("expected newOffset %d, got %d", tt.expectedOffset, newOffset)
			}

			// Ensure no additional data was decoded
			if newOffset < len(tt.input) && tt.input[newOffset] != 0 {
				t.Errorf("unexpected data decoded beyond newOffset")
			}
		})
	}
}

// floatEqual checks if two float64 values are approximately equal.
func floatEqual(a, b float64) bool {
	const epsilon = 1e-7
	return math.Abs(a-b) < epsilon
}
