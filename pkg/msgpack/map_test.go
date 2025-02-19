package msgpack

import (
	"bytes"
	"errors"
	"testing"
)

func TestAppendMapHeader(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected []byte
	}{
		{"Small Map", 5, []byte{0x85}},
		{"16-bit Map", 256, []byte{0xde, 0x01, 0x00}},
		{"32-bit Map", 65536, []byte{0xdf, 0x00, 0x01, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendMapHeader(nil, tt.n)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadMapHeader(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedLength int
		expectedOffset int
		expectedErr    error
	}{
		{"Small Map", []byte{0x85}, 0, 5, 1, nil},
		{"16-bit Map", []byte{0xde, 0x01, 0x00}, 0, 256, 3, nil},
		{"32-bit Map", []byte{0xdf, 0x00, 0x01, 0x00, 0x00}, 0, 65536, 5, nil},
		{"Invalid Header", []byte{0x75}, 0, 0, 0, ErrInvalidHeaderByte},
		{"Empty Input", []byte{}, 0, 0, 0, ErrShortBuffer},
		{"Offset Out of Bounds", []byte{0x85}, 2, 0, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length, newOffset, err := ReadMapHeader(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify length and newOffset
			if length != tt.expectedLength {
				t.Errorf("expected length %d, got %d", tt.expectedLength, length)
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
