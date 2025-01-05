package msgpack

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAppendNil(t *testing.T) {
	t.Run("Append Nil", func(t *testing.T) {
		result := AppendNil(nil)
		expected := []byte{0xc0}

		if !bytes.Equal(result, expected) {
			t.Errorf("expected %x, got %x", expected, result)
		}
	})
}

func TestReadNil(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedOffset int
		expectedErr    error
	}{
		{"Valid Nil", []byte{0xc0}, 0, 1, nil},
		{"Invalid Header", []byte{0xc1}, 0, 0, fmt.Errorf("expected nil (0xc0), got 0xc1")},
		{"Empty Input", []byte{}, 0, 0, ErrShortBuffer},
		{"Offset Out of Bounds", []byte{0xc0}, 2, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newOffset, err := ReadNil(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify new offset
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
