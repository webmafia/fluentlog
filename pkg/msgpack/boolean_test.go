package msgpack

import (
	"errors"
	"testing"
)

func TestAppendBool(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected []byte
	}{
		{"True", true, []byte{0xc3}},
		{"False", false, []byte{0xc2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendBool(nil, tt.input)
			if len(result) != 1 || result[0] != tt.expected[0] {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadBool(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedValue  bool
		expectedOffset int
		expectedErr    error
	}{
		{"True", []byte{0xc3}, 0, true, 1, nil},
		{"False", []byte{0xc2}, 0, false, 1, nil},
		{"Invalid Header", []byte{0xc1}, 0, false, 0, ErrInvalidHeaderByte},
		{"Empty Input", []byte{}, 0, false, 0, ErrShortBuffer},
		{"Offset Out of Bounds", []byte{0xc3}, 2, false, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, newOffset, err := ReadBool(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify decoded value and offset
			if value != tt.expectedValue {
				t.Errorf("expected value %v, got %v", tt.expectedValue, value)
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
