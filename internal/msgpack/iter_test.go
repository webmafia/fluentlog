package msgpack

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

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

	fmt.Println("---")

	r := bytes.NewReader(data)
	iter := NewIterator(r, buffer.NewBuffer(4096), 4096)

	for iter.Next() {
		fmt.Println(iter.Value())

		if iter.Type() == types.Array {
			iter.Skip()
		}
	}

	fmt.Println(iter.Error())

	// Output:
	//
	// [131 170 115 105 109 112 108 101 95 107 101 121 172 115 105 109 112 108 101 95 118 97 108 117 101 172 110 101 115 116 101 100 95 97 114 114 97 121 147 1 2 147 3 4 5 170 110 101 115 116 101 100 95 109 97 112 130 169 105 110 110 101 114 95 107 101 121 146 195 194 169 102 108 111 97 116 95 107 101 121 203 64 9 33 249 240 27 134 110]
}

func BenchmarkIterator(b *testing.B) {
	msg := buildComplexMessage()
	iter := NewIterator(bytes.NewReader(msg), buffer.NewBuffer(4096), 4096)

	b.ResetTimer()

	var i int

	for range b.N {
		iter.ResetBytes(msg)
		i = 0

		for iter.Next() {
			i++
			_ = iter.Type()
		}
	}

	b.ReportMetric(float64(b.Elapsed())/float64(i)/float64(b.N), "ns/field")
}
