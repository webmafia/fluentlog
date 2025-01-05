package msgpack

import (
	"bytes"
	"errors"
	"testing"
)

func TestAppendArrayHeader(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected []byte
	}{
		{"Small Array", 5, []byte{0x95}},
		{"16-bit Array", 256, []byte{0xdc, 0x01, 0x00}},
		{"32-bit Array", 65536, []byte{0xdd, 0x00, 0x01, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendArrayHeader(nil, tt.n)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadArrayHeader(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedLen    int
		expectedOffset int
		expectedErr    error
	}{
		{"Small Array", []byte{0x95}, 0, 5, 1, nil},
		{"16-bit Array", []byte{0xdc, 0x01, 0x00}, 0, 256, 3, nil},
		{"32-bit Array", []byte{0xdd, 0x00, 0x01, 0x00, 0x00}, 0, 65536, 5, nil},
		{"Invalid Type", []byte{0x85}, 0, 0, 0, ErrInvalidHeaderByte},
		{"Empty Input", []byte{}, 0, 0, 0, ErrShortBuffer},
		{"Offset Out of Bounds", []byte{0x95}, 2, 0, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length, newOffset, err := ReadArrayHeader(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify length and newOffset
			if length != tt.expectedLen {
				t.Errorf("expected length %d, got %d", tt.expectedLen, length)
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
