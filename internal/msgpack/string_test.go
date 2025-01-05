package msgpack

import (
	"bytes"
	"errors"
	"testing"
)

func TestAppendString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{"Small String", "abc", append([]byte{0xa3}, []byte("abc")...)},
		{"Str8", string(make([]byte, 255)), append([]byte{0xd9, 0xff}, make([]byte, 255)...)},
		{"Str16", string(make([]byte, 256)), append([]byte{0xda, 0x01, 0x00}, make([]byte, 256)...)},
		{"Str32", string(make([]byte, 65536)), append([]byte{0xdb, 0x00, 0x01, 0x00, 0x00}, make([]byte, 65536)...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendString(nil, tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedValue  string
		expectedOffset int
		expectedErr    error
	}{
		{"Small String", []byte{0xa3, 'a', 'b', 'c'}, 0, "abc", 4, nil},
		{"Str8", append([]byte{0xd9, 0x03}, []byte("xyz")...), 0, "xyz", 5, nil},
		{"Str16", append([]byte{0xda, 0x00, 0x03}, []byte("def")...), 0, "def", 5, nil},
		{"Invalid Header", []byte{0x95, 0x00}, 0, "", 0, ErrInvalidHeaderByte},
		{"Empty Input", []byte{}, 0, "", 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, newOffset, err := ReadString(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify decoded value and offset
			if result != tt.expectedValue {
				t.Errorf("expected value %q, got %q", tt.expectedValue, result)
			}
			if newOffset != tt.expectedOffset {
				t.Errorf("expected newOffset %d, got %d", tt.expectedOffset, newOffset)
			}
		})
	}
}

func TestReadStringCopy(t *testing.T) {
	t.Run("String Copy", func(t *testing.T) {
		input := []byte{0xa3, 'a', 'b', 'c'}
		result, _, err := ReadStringCopy(input, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "abc" {
			t.Errorf("expected value %q, got %q", "abc", result)
		}

		// Ensure a copy was made
		input[1] = 'z'
		if result == string(input[1:4]) {
			t.Errorf("expected a copy of the string, but result shares memory with input")
		}
	})
}

func TestAppendTextAppender(t *testing.T) {
	t.Run("Text Appender", func(t *testing.T) {
		mockAppender := &mockTextAppender{data: "appended text"}
		expected := AppendString(nil, mockAppender.data)
		result := AppendTextAppender(nil, mockAppender)

		if !bytes.Equal(result, expected) {
			t.Errorf("expected %x, got %x", expected, result)
		}
	})
}

func TestAppendStringUnknownLength(t *testing.T) {
	t.Run("Unknown Length", func(t *testing.T) {
		data := "unknown length"
		fn := func(dst []byte) []byte {
			return append(dst, data...)
		}
		expected := AppendString(nil, data)
		result := AppendStringUnknownLength(nil, fn)

		if !bytes.Equal(result, expected) {
			t.Errorf("expected %x, got %x", expected, result)
		}
	})
}

// mockTextAppender is a mock implementation of internal.TextAppender.
type mockTextAppender struct {
	data string
}

func (m *mockTextAppender) AppendText(dst []byte) ([]byte, error) {
	return append(dst, m.data...), nil
}
