package msgpack

import (
	"fmt"
	"testing"
)

func ExampleValue_Array() {
	var buf Value

	buf = AppendArray(buf, 3)
	buf = AppendString(buf, "foo")
	buf = AppendString(buf, "bar")
	buf = AppendString(buf, "baz")

	fmt.Println(buf.Len())

	for v := range buf.Array() {
		fmt.Println(v.String())
	}

	// Output: TODO
}

func ExampleValue_Map() {
	var buf Value

	buf = AppendMap(buf, 3)
	buf = AppendString(buf, "foo")
	buf = AppendInt(buf, 123)
	buf = AppendString(buf, "bar")
	buf = AppendInt(buf, 456)
	buf = AppendString(buf, "baz")
	buf = AppendInt(buf, 789)

	fmt.Println(buf.Len())

	for k, v := range buf.Map() {
		fmt.Println(k.String(), v.String())
	}

	// Output: TODO
}

func BenchmarkValueLen(b *testing.B) {
	var v Value
	v = AppendArray(v, 3)

	b.ResetTimer()

	for range b.N {
		_ = v.Len()
	}
}

func TestValue_BytesLen(t *testing.T) {
	tests := []struct {
		input    Value
		expected int
		desc     string
	}{
		// Positive FixInt (head only)
		{Value{0x00}, 1, "Positive FixInt: 0"},
		{Value{0x7f}, 1, "Positive FixInt: 127"},

		// Negative FixInt (head only)
		{Value{0xe0}, 1, "Negative FixInt: -32"},
		{Value{0xff}, 1, "Negative FixInt: -1"},

		// FixStr (head + body)
		{Value{0xa0}, 1, "FixStr: empty string"},
		{Value{0xa5, 'h', 'e', 'l', 'l', 'o'}, 6, "FixStr: 'hello'"},

		// FixArray (head only, excludes body)
		{Value{0x90}, 1, "FixArray: empty array"},
		{Value{0x93, 0x01, 0x02, 0x03}, 1, "FixArray: array with 3 elements"},

		// FixMap (head only, excludes body)
		{Value{0x80}, 1, "FixMap: empty map"},
		{Value{0x82, 0xa1, 'a', 0x01, 0xa1, 'b', 0x02}, 1, "FixMap: map with 2 key-value pairs"},

		// Nil (head only)
		{Value{0xc0}, 1, "Nil value"},

		// Bool (head only)
		{Value{0xc2}, 1, "Bool: false"},
		{Value{0xc3}, 1, "Bool: true"},

		// Binary (head + body)
		{Value{0xc4, 0x03, 0x01, 0x02, 0x03}, 5, "Bin8 with 3 bytes"},
		{Value{0xc5, 0x00, 0x03, 0x01, 0x02, 0x03}, 6, "Bin16 with 3 bytes"},

		// Float (head only)
		{Value{0xca, 0x41, 0x20, 0x00, 0x00}, 5, "Float32: 10.0"},
		{Value{0xcb, 0x40, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 9, "Float64: 10.0"},

		// Unsigned integers (head only)
		{Value{0xcc, 0xff}, 2, "Uint8: 255"},
		{Value{0xcd, 0x01, 0x00}, 3, "Uint16: 256"},
		{Value{0xce, 0x00, 0x00, 0x01, 0x00}, 5, "Uint32: 256"},

		// Signed integers (head only)
		{Value{0xd0, 0xff}, 2, "Int8: -1"},
		{Value{0xd1, 0xff, 0xfe}, 3, "Int16: -2"},
		{Value{0xd2, 0xff, 0xff, 0xff, 0xfe}, 5, "Int32: -2"},

		// String types (head + body)
		{Value{0xd9, 0x05, 'h', 'e', 'l', 'l', 'o'}, 7, "Str8: 'hello'"},
		{Value{0xda, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}, 8, "Str16: 'hello'"},

		// Array types (head only, excludes body)
		{Value{0xdc, 0x00, 0x03, 0x01, 0x02, 0x03}, 3, "Array16 with 3 elements"},
		{Value{0xdd, 0x00, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03}, 5, "Array32 with 3 elements"},

		// Map types (head only, excludes body)
		{Value{0xde, 0x00, 0x02, 0xa1, 'a', 0x01, 0xa1, 'b', 0x02}, 3, "Map16 with 2 key-value pairs"},
		{Value{0xdf, 0x00, 0x00, 0x00, 0x02, 0xa1, 'a', 0x01, 0xa1, 'b', 0x02}, 5, "Map32 with 2 key-value pairs"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.input.BytesLen(); got != tt.expected {
				t.Errorf("Value.BytesLen() = %d; want %d", got, tt.expected)
			}
		})
	}
}
