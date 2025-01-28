package fuzz

import (
	"github.com/webmafia/fluentlog/internal/msgpack"
)

// BuildComplexMessage creates a deep, complex MessagePack message using Append* functions.
func buildComplexMessage() []byte {
	var data []byte

	// Example: A MessagePack map with nested structures
	data = msgpack.AppendMapHeader(data, 3) // Map with 3 key-value pairs

	// Key 1: "simple_key" -> "simple_value"
	data = msgpack.AppendString(data, "simple_key")
	data = msgpack.AppendString(data, "simple_value")

	// Key 2: "nested_array" -> [1, 2, [3, 4, 5]]
	data = msgpack.AppendString(data, "nested_array")
	data = msgpack.AppendArrayHeader(data, 3)
	data = msgpack.AppendInt(data, 1)
	data = msgpack.AppendInt(data, 2)
	data = msgpack.AppendArrayHeader(data, 3)
	data = msgpack.AppendInt(data, 3)
	data = msgpack.AppendInt(data, 4)
	data = msgpack.AppendInt(data, 5)

	// Key 3: "nested_map" -> { "inner_key": [true, false], "float_key": 3.14159 }
	data = msgpack.AppendString(data, "nested_map")
	data = msgpack.AppendMapHeader(data, 2)
	data = msgpack.AppendString(data, "inner_key")
	data = msgpack.AppendArrayHeader(data, 2)
	data = msgpack.AppendBool(data, true)
	data = msgpack.AppendBool(data, false)
	data = msgpack.AppendString(data, "float_key")
	data = msgpack.AppendFloat(data, 3.14159)

	return data
}
