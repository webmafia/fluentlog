package msgpack

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func BenchmarkIterator(b *testing.B) {
	benchmarks := []struct {
		input       []byte
		description string
	}{
		{[]byte{0xc0}, "Nil type"},
		{[]byte{0xca, 0x40, 0x49, 0x0f, 0xdb}, "Float32 type"},
		{[]byte{0xcc, 0xff}, "Uint8 type"},
		{[]byte{0xd9, 0x05, 'h', 'e', 'l', 'l', 'o'}, "Str8 type with 'hello'"},
		{[]byte{0xde, 0x00, 0x02}, "Map16 with 2 key-value pairs"},
		{[]byte{0xdc, 0x00, 0x03}, "Array16 with 3 elements"},
		{[]byte{0xdf, 0x00, 0x00, 0x00, 0x04}, "Map32 with 4 key-value pairs"},
	}

	b.Run("Baseline", func(b *testing.B) {
		r := bytes.NewBuffer(nil)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			r.Reset()
			r.Write([]byte{0xdf, 0x00, 0x00, 0x00, 0x04})
		}
	})

	for _, bm := range benchmarks {
		b.Run(bm.description, func(b *testing.B) {
			r := bytes.NewBuffer(nil)
			r2 := NewIterator(r, buffer.NewBuffer(64), 4096)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r.Reset()
				r.Write(bm.input)
				r2.Reset(r)
				_, _ = r2.Next()
				_ = r2.ReadBinary()
			}
		})
	}
}

// BuildComplexMessage creates a deep, complex MessagePack message using Append* functions.
func buildComplexMessage() []byte {
	var data []byte

	// Example: A MessagePack map with nested structures
	data = AppendMapHeader(data, 3) // Map with 3 key-value pairs

	// Key 1: "simple_key" -> "simple_value"
	data = AppendString(data, "simple_key")
	data = AppendString(data, "simple_value")

	// Key 2: "nested_array" -> [1, 2, [3, 4, 5]]
	data = AppendString(data, "nested_array")
	data = AppendArrayHeader(data, 3)
	data = AppendInt(data, 1)
	data = AppendInt(data, 2)
	data = AppendArrayHeader(data, 3)
	data = AppendInt(data, 3)
	data = AppendInt(data, 4)
	data = AppendInt(data, 5)

	// Key 3: "nested_map" -> { "inner_key": [true, false], "float_key": 3.14159 }
	data = AppendString(data, "nested_map")
	data = AppendMapHeader(data, 2)
	data = AppendString(data, "inner_key")
	data = AppendArrayHeader(data, 2)
	data = AppendBool(data, true)
	data = AppendBool(data, false)
	data = AppendString(data, "float_key")
	data = AppendFloat(data, 3.14159)

	return data
}

func Example_buildComplexMessage() {
	data := buildComplexMessage()
	fmt.Println(data)

	// Output:
	//
	// [131 170 115 105 109 112 108 101 95 107 101 121 172 115 105 109 112 108 101 95 118 97 108 117 101 172 110 101 115 116 101 100 95 97 114 114 97 121 147 1 2 147 3 4 5 170 110 101 115 116 101 100 95 109 97 112 130 169 105 110 110 101 114 95 107 101 121 146 195 194 169 102 108 111 97 116 95 107 101 121 203 64 9 33 249 240 27 134 110]
}

func TestIterator_ComplexMessage(t *testing.T) {
	data := buildComplexMessage()                              // Build the complex message
	buf := buffer.NewBuffer(len(data))                         // Create buffer
	iter := NewIterator(bytes.NewReader(data), buf, len(data)) // Initialize iterator

	// Step into the root map
	if typ, length := iter.Next(); typ != types.Map || length != 3 {
		t.Fatalf("expected type Map with length 3, got type %v, length %d", typ, length)
	}

	// Key 1: Validate "simple_key" -> "simple_value"
	if typ, _ := iter.Next(); typ != types.Str {
		t.Fatalf("expected type Str for key 'simple_key', got %v", typ)
	}
	if key := iter.ReadString(); key != "simple_key" {
		t.Fatalf("expected key 'simple_key', got %s", key)
	}
	if typ, _ := iter.Next(); typ != types.Str {
		t.Fatalf("expected type Str for value 'simple_value', got %v", typ)
	}
	if value := iter.ReadString(); value != "simple_value" {
		t.Fatalf("expected value 'simple_value', got %s", value)
	}

	// Key 2: Validate "nested_array" -> [1, 2, [3, 4, 5]]
	if typ, _ := iter.Next(); typ != types.Str {
		t.Fatalf("expected type Str for key 'nested_array', got %v", typ)
	}
	if key := iter.ReadString(); key != "nested_array" {
		t.Fatalf("expected key 'nested_array', got %s", key)
	}
	if typ, length := iter.Next(); typ != types.Array || length != 3 {
		t.Fatalf("expected type Array with length 3 for value 'nested_array', got type %v, length %d", typ, length)
	}

	// Step into array
	if typ, _ := iter.Next(); typ != types.Uint {
		t.Fatalf("expected type Uint for value 1, got %v", typ)
	}
	if value := iter.ReadInt(); value != 1 {
		t.Fatalf("expected value 1, got %d", value)
	}
	if typ, _ := iter.Next(); typ != types.Uint {
		t.Fatalf("expected type Uint for value 2, got %v", typ)
	}
	if value := iter.ReadInt(); value != 2 {
		t.Fatalf("expected value 2, got %d", value)
	}
	if typ, length := iter.Next(); typ != types.Array || length != 3 {
		t.Fatalf("expected type Array with length 3 for nested array, got type %v, length %d", typ, length)
	}

	// Step into nested array
	if typ, _ := iter.Next(); typ != types.Uint {
		t.Fatalf("expected type Uint for value 3, got %v", typ)
	}
	if value := iter.ReadInt(); value != 3 {
		t.Fatalf("expected value 3, got %d", value)
	}
	if typ, _ := iter.Next(); typ != types.Uint {
		t.Fatalf("expected type Uint for value 4, got %v", typ)
	}
	if value := iter.ReadInt(); value != 4 {
		t.Fatalf("expected value 4, got %d", value)
	}
	if typ, _ := iter.Next(); typ != types.Uint {
		t.Fatalf("expected type Uint for value 5, got %v", typ)
	}
	if value := iter.ReadInt(); value != 5 {
		t.Fatalf("expected value 5, got %d", value)
	}

	// Key 3: Validate "nested_map" -> { "inner_key": [true, false], "float_key": 3.14159 }
	if typ, _ := iter.Next(); typ != types.Str {
		t.Fatalf("expected type Str for key 'nested_map', got %v", typ)
	}
	if key := iter.ReadString(); key != "nested_map" {
		t.Fatalf("expected key 'nested_map', got %s", key)
	}
	if typ, length := iter.Next(); typ != types.Map || length != 2 {
		t.Fatalf("expected type Map with length 2 for value 'nested_map', got type %v, length %d", typ, length)
	}

	// Step into map
	if typ, _ := iter.Next(); typ != types.Str {
		t.Fatalf("expected type Str for key 'inner_key', got %v", typ)
	}
	if key := iter.ReadString(); key != "inner_key" {
		t.Fatalf("expected key 'inner_key', got %s", key)
	}
	if typ, length := iter.Next(); typ != types.Array || length != 2 {
		t.Fatalf("expected type Array with length 2 for value 'inner_key', got type %v, length %d", typ, length)
	}

	// Step into inner array
	if typ, _ := iter.Next(); typ != types.Bool {
		t.Fatalf("expected type Bool for value true, got %v", typ)
	}
	if value := iter.ReadBool(); !value {
		t.Fatalf("expected value true, got false")
	}
	if typ, _ := iter.Next(); typ != types.Bool {
		t.Fatalf("expected type Bool for value false, got %v", typ)
	}
	if value := iter.ReadBool(); value {
		t.Fatalf("expected value false, got true")
	}

	// Continue in map
	if typ, _ := iter.Next(); typ != types.Str {
		t.Fatalf("expected type Str for key 'float_key', got %v", typ)
	}
	if key := iter.ReadString(); key != "float_key" {
		t.Fatalf("expected key 'float_key', got %s", key)
	}
	if typ, _ := iter.Next(); typ != types.Float {
		t.Fatalf("expected type Float for value 3.14159, got %v", typ)
	}
	if value := iter.ReadFloat(); value != 3.14159 {
		t.Fatalf("expected value 3.14159, got %f", value)
	}
}
func FuzzIterator_ComplexMessage(f *testing.F) {
	// Seed the fuzzer with the complex message
	complexMessage := buildComplexMessage()
	f.Add(complexMessage)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create buffer and iterator
		buf := buffer.NewBuffer(len(data))
		iter := NewIterator(bytes.NewReader(data), buf, len(data))

		// Iterate through the data
		for {
			// Call Next() to advance the iterator
			typ, _ := iter.Next()

			// Stop iteration if type is Nil (end of data)
			if typ == types.Nil {
				break
			}

			// Check for any errors after calling Next()
			if err := iter.Error(); err != nil {
				t.Fatalf("unexpected iterator error during Next(): %v", err)
			}

			// Handle each type based on the MessagePack spec
			switch typ {
			case types.Bool:
				iter.ReadBool()
			case types.Int, types.Uint:
				iter.ReadInt()
			case types.Float:
				iter.ReadFloat()
			case types.Str:
				iter.ReadString()
			case types.Bin:
				iter.ReadBinary()
			case types.Array:
				// Step into the array and consume all elements
				for {
					typ, _ := iter.Next()
					if typ == types.Nil || iter.Error() != nil {
						break
					}
				}
			case types.Map:
				// Step into the map and consume all key-value pairs
				for {
					typ, _ := iter.Next()
					if typ == types.Nil || iter.Error() != nil {
						break
					}
				}
			default:
				// Unsupported or unexpected type
				t.Fatalf("unexpected type: %v", typ)
			}
		}

		// Ensure no lingering errors
		if err := iter.Error(); err != nil {
			t.Fatalf("iterator ended with error: %v", err)
		}
	})
}
