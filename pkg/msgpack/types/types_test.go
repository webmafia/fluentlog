package types

import (
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		input           byte
		expectedType    Type
		expectedLength  int
		expectedIsValue bool
		description     string
	}{
		// Positive FixInt (0x00 to 0x7f)
		{0x00, Uint, 0, true, "Positive FixInt: smallest value (0)"},
		{0x7f, Uint, 0, true, "Positive FixInt: largest value (127)"},

		// Negative FixInt (0xe0 to 0xff)
		{0xe0, Int, 0, true, "Negative FixInt: smallest value (-32)"},
		{0xff, Int, 0, true, "Negative FixInt: largest value (-1)"},

		// FixStr (0xa0 to 0xbf)
		{0xa0, Str, 0, true, "FixStr: empty string"},
		{0xa5, Str, 5, true, "FixStr: string of length 5"},
		{0xbf, Str, 31, true, "FixStr: largest string length (31)"},

		// FixMap (0x80 to 0x8f)
		{0x80, Map, 0, true, "FixMap: empty map"},
		{0x85, Map, 5, true, "FixMap: map with 5 key-value pairs"},
		{0x8f, Map, 15, true, "FixMap: largest map with 15 key-value pairs"},

		// FixArray (0x90 to 0x9f)
		{0x90, Array, 0, true, "FixArray: empty array"},
		{0x93, Array, 3, true, "FixArray: array with 3 elements"},
		{0x9f, Array, 15, true, "FixArray: largest array with 15 elements"},

		// Fixed types
		{0xc0, Nil, 0, true, "Nil"},
		{0xc2, Bool, 0, true, "Bool: false"},
		{0xc3, Bool, 0, true, "Bool: true"},

		// Binary types
		{0xc4, Bin, 1, false, "Bin8"},
		{0xc5, Bin, 2, false, "Bin16"},
		{0xc6, Bin, 4, false, "Bin32"},

		// Extension types
		{0xc7, Ext, 1, false, "Ext8"},
		{0xc8, Ext, 2, false, "Ext16"},
		{0xc9, Ext, 4, false, "Ext32"},

		// Float types
		{0xca, Float, 4, true, "Float32"},
		{0xcb, Float, 8, true, "Float64"},

		// Unsigned integers
		{0xcc, Uint, 1, true, "Uint8"},
		{0xcd, Uint, 2, true, "Uint16"},
		{0xce, Uint, 4, true, "Uint32"},
		{0xcf, Uint, 8, true, "Uint64"},

		// Signed integers
		{0xd0, Int, 1, true, "Int8"},
		{0xd1, Int, 2, true, "Int16"},
		{0xd2, Int, 4, true, "Int32"},
		{0xd3, Int, 8, true, "Int64"},

		// String types
		{0xd9, Str, 1, false, "Str8"},
		{0xda, Str, 2, false, "Str16"},
		{0xdb, Str, 4, false, "Str32"},

		// Array types
		{0xdc, Array, 2, false, "Array16"},
		{0xdd, Array, 4, false, "Array32"},

		// Map types
		{0xde, Map, 2, false, "Map16"},
		{0xdf, Map, 4, false, "Map32"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			typ, length, isValue := Get(tt.input)
			if typ != tt.expectedType || length != tt.expectedLength || isValue != tt.expectedIsValue {
				t.Errorf("Get(0x%x) = (%v, %d, %v); want (%v, %d, %v)",
					tt.input, typ, length, isValue, tt.expectedType, tt.expectedLength, tt.expectedIsValue)
			}
		})
	}
}
