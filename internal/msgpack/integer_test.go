package msgpack

import (
	"bytes"
	"errors"
	"testing"
)

func TestAppendInt(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected []byte
	}{
		{"Positive FixInt", 42, []byte{0x2a}},
		{"Negative FixInt", -5, []byte{0xfb}},
		{"Int8", -128, []byte{0xd0, 0x80}},
		{"Int16", -32768, []byte{0xd1, 0x80, 0x00}},
		{"Int32", -2147483648, []byte{0xd2, 0x80, 0x00, 0x00, 0x00}},
		{"Int64", -9223372036854775808, []byte{0xd3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendInt(nil, tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadInt(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedValue  int64
		expectedOffset int
		expectedErr    error
	}{
		{"Positive FixInt", []byte{0x2a}, 0, 42, 1, nil},
		{"Negative FixInt", []byte{0xfb}, 0, -5, 1, nil},
		{"Int8", []byte{0xd0, 0x80}, 0, -128, 2, nil},
		{"Int16", []byte{0xd1, 0x80, 0x00}, 0, -32768, 3, nil},
		{"Int32", []byte{0xd2, 0x80, 0x00, 0x00, 0x00}, 0, -2147483648, 5, nil},
		{"Int64", []byte{0xd3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 0, -9223372036854775808, 9, nil},
		{"Invalid Header", []byte{0xcc, 0x00}, 0, 0, 0, ErrInvalidHeaderByte},
		{"Short Buffer", []byte{0xd1, 0x80}, 0, 0, 0, ErrShortBuffer},
		{"Empty Input", []byte{}, 0, 0, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, newOffset, err := ReadInt(tt.input, tt.offset)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if value != tt.expectedValue {
				t.Errorf("expected value %d, got %d", tt.expectedValue, value)
			}
			if newOffset != tt.expectedOffset {
				t.Errorf("expected newOffset %d, got %d", tt.expectedOffset, newOffset)
			}
		})
	}
}

func TestAppendUint(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected []byte
	}{
		{"Positive FixInt", 42, []byte{0x2a}},
		{"Uint8", 255, []byte{0xcc, 0xff}},
		{"Uint16", 65535, []byte{0xcd, 0xff, 0xff}},
		{"Uint32", 4294967295, []byte{0xce, 0xff, 0xff, 0xff, 0xff}},
		{"Uint64", 18446744073709551615, []byte{0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendUint(nil, tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadUint(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedValue  uint64
		expectedOffset int
		expectedErr    error
	}{
		{"Positive FixInt", []byte{0x2a}, 0, 42, 1, nil},
		{"Uint8", []byte{0xcc, 0xff}, 0, 255, 2, nil},
		{"Uint16", []byte{0xcd, 0xff, 0xff}, 0, 65535, 3, nil},
		{"Uint32", []byte{0xce, 0xff, 0xff, 0xff, 0xff}, 0, 4294967295, 5, nil},
		{"Uint64", []byte{0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 0, 18446744073709551615, 9, nil},
		{"Invalid Header", []byte{0xd0, 0x00}, 0, 0, 0, ErrInvalidHeaderByte},
		{"Short Buffer", []byte{0xcd, 0xff}, 0, 0, 0, ErrShortBuffer},
		{"Empty Input", []byte{}, 0, 0, 0, ErrShortBuffer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, newOffset, err := ReadUint(tt.input, tt.offset)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if value != tt.expectedValue {
				t.Errorf("expected value %d, got %d", tt.expectedValue, value)
			}
			if newOffset != tt.expectedOffset {
				t.Errorf("expected newOffset %d, got %d", tt.expectedOffset, newOffset)
			}
		})
	}
}
