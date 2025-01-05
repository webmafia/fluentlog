package msgpack

import (
	"bytes"
	"errors"
	"testing"
)

func TestAppendBinary(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
	}{
		{"Empty Data", []byte{}, []byte{0xc4, 0x00}},
		{"Small Data", []byte{0x01, 0x02, 0x03}, []byte{0xc4, 0x03, 0x01, 0x02, 0x03}},
		{"16-bit Data", make([]byte, 256), append([]byte{0xc5, 0x01, 0x00}, make([]byte, 256)...)},
		{"32-bit Data", make([]byte, 65536), append([]byte{0xc6, 0x00, 0x01, 0x00, 0x00}, make([]byte, 65536)...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendBinary(nil, tt.data)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadBinary(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedData   []byte
		expectedOffset int
		expectedErr    error
	}{
		{"Empty Data", []byte{0xc4, 0x00}, 0, []byte{}, 2, nil},
		{"Small Data", []byte{0xc4, 0x03, 0x01, 0x02, 0x03}, 0, []byte{0x01, 0x02, 0x03}, 5, nil},
		{"16-bit Data", append([]byte{0xc5, 0x01, 0x00}, make([]byte, 256)...), 0, make([]byte, 256), 259, nil},
		{"32-bit Data", append([]byte{0xc6, 0x00, 0x01, 0x00, 0x00}, make([]byte, 65536)...), 0, make([]byte, 65536), 65541, nil},
		{"Invalid Header", []byte{0xd4, 0x00}, 0, nil, 0, ErrInvalidHeaderByte},
		{"Short Buffer", []byte{0xc4, 0x05, 0x01, 0x02}, 0, nil, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, newOffset, err := ReadBinary(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify decoded data and offset
			if !bytes.Equal(data, tt.expectedData) {
				t.Errorf("expected data %x, got %x", tt.expectedData, data)
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

func TestAppendBinaryAppender(t *testing.T) {
	t.Run("Binary Appender", func(t *testing.T) {
		mockAppender := &mockBinaryAppender{
			data: []byte{0x01, 0x02, 0x03},
		}
		expected := AppendBinary(nil, mockAppender.data)
		result := AppendBinaryAppender(nil, mockAppender)

		if !bytes.Equal(result, expected) {
			t.Errorf("expected %x, got %x", expected, result)
		}
	})
}

func TestAppendBinaryUnknownLength(t *testing.T) {
	t.Run("Unknown Length", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03}
		fn := func(dst []byte) []byte {
			return append(dst, data...)
		}
		expected := AppendBinary(nil, data)
		result := AppendBinaryUnknownLength(nil, fn)

		if !bytes.Equal(result, expected) {
			t.Errorf("expected %x, got %x", expected, result)
		}
	})
}

// mockBinaryAppender is a mock implementation of internal.BinaryAppender.
type mockBinaryAppender struct {
	data []byte
}

func (m *mockBinaryAppender) AppendBinary(dst []byte) ([]byte, error) {
	return append(dst, m.data...), nil
}
