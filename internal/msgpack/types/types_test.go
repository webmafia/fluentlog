package types

import (
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		input    byte
		expected Type
	}{
		{0x00, Uint},  // Positive FixInt
		{0x7f, Uint},  // Positive FixInt
		{0xe0, Int},   // Negative FixInt
		{0xff, Int},   // Negative FixInt
		{0xa0, Str},   // FixStr
		{0xbf, Str},   // FixStr
		{0x80, Map},   // FixMap
		{0x8f, Map},   // FixMap
		{0x90, Array}, // FixArray
		{0x9f, Array}, // FixArray
		{0xc0, Nil},   // nil
		{0xc2, Bool},  // false
		{0xc3, Bool},  // true
		{0xca, Float}, // float32
		{0xcb, Float}, // float64
		{0xcc, Uint},  // uint8
		{0xd0, Int},   // int8
		{0xd9, Str},   // str8
		{0xdc, Array}, // array16
		{0xde, Map},   // map16
		// Add more cases as needed
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := Get(tt.input)
			if result != tt.expected {
				t.Errorf("Get(0x%x) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetLength(t *testing.T) {
	tests := []struct {
		input           byte
		expectedLength  int
		expectedIsValue bool
	}{
		{0x00, 0, true},  // Positive FixInt
		{0xe0, 0, true},  // Negative FixInt
		{0xa5, 5, true},  // FixStr with length 5
		{0x85, 10, true}, // FixMap with 5 key-value pairs
		{0x93, 3, true},  // FixArray with 3 elements
		{0xc0, 0, true},  // nil
		{0xc4, 1, false}, // bin8
		{0xca, 4, true},  // float32
		{0xd9, 1, false}, // str8
		{0xdc, 2, false}, // array16
		{0xde, 2, false}, // map16
		// Add more cases as needed
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			length, isValue := GetLength(tt.input)
			if length != tt.expectedLength || isValue != tt.expectedIsValue {
				t.Errorf("GetLength(0x%x) = (%v, %v); want (%v, %v)",
					tt.input, length, isValue, tt.expectedLength, tt.expectedIsValue)
			}
		})
	}
}
