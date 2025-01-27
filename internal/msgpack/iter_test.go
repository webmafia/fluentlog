package msgpack

import (
	"bytes"
	"fmt"
	"testing"
	"time"
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

	// Output:
	//
	// [131 170 115 105 109 112 108 101 95 107 101 121 172 115 105 109 112 108 101 95 118 97 108 117 101 172 110 101 115 116 101 100 95 97 114 114 97 121 147 1 2 147 3 4 5 170 110 101 115 116 101 100 95 109 97 112 130 169 105 110 110 101 114 95 107 101 121 146 195 194 169 102 108 111 97 116 95 107 101 121 203 64 9 33 249 240 27 134 110]
}

func Example_iterateComplexMessage() {
	data := buildComplexMessage()
	iter := NewIterator(nil)
	iter.ResetBytes(data)

	for iter.Next() {
		fmt.Println(iter.Type())
		// iter.Skip()
	}

	// Output: TODO
}

func BenchmarkIterator(b *testing.B) {
	msg := buildComplexMessage()
	iter := NewIterator(bytes.NewReader(msg))

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

func BenchmarkIterator_Next(b *testing.B) {
	b.Run("baseline", func(b *testing.B) {
		data := AppendArrayHeader(nil, 10)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
		}
	})

	b.Run("ArrayHeader", func(b *testing.B) {
		data := AppendArrayHeader(nil, 10)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("Binary", func(b *testing.B) {
		data := AppendBinary(nil, []byte("example binary data"))
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("Bool", func(b *testing.B) {
		data := AppendBool(nil, true)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("Float", func(b *testing.B) {
		data := AppendFloat(nil, 3.14159)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("Int", func(b *testing.B) {
		data := AppendInt(nil, -123456)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("Uint", func(b *testing.B) {
		data := AppendUint(nil, 123456)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("MapHeader", func(b *testing.B) {
		data := AppendMapHeader(nil, 5)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("Nil", func(b *testing.B) {
		data := AppendNil(nil)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	b.Run("String", func(b *testing.B) {
		data := AppendString(nil, "example string")
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			iter.ResetBytes(data)
			_ = iter.Next()
		}
	})

	for format, formatName := range tsFormatStrings {
		b.Run("Timestamp_"+formatName, func(b *testing.B) {
			data := AppendTimestamp(nil, time.Unix(1672531200, 500000000), TsFormat(format))
			iter := NewIterator(nil)
			iter.ResetBytes(data)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				iter.ResetBytes(data)
				_ = iter.Next()
			}
		})
	}
}

func BenchmarkIterator_Read(b *testing.B) {
	b.Run("ArrayHeader", func(b *testing.B) {
		data := AppendArrayHeader(nil, 10)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Len()
		}
	})

	b.Run("Binary", func(b *testing.B) {
		data := AppendBinary(nil, []byte("example binary data"))
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Bin()
		}
	})

	b.Run("Bool", func(b *testing.B) {
		data := AppendBool(nil, true)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Bool()
		}
	})

	b.Run("Float", func(b *testing.B) {
		data := AppendFloat(nil, 3.14159)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Float()
		}
	})

	b.Run("Int", func(b *testing.B) {
		data := AppendInt(nil, -123456)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Int()
		}
	})

	b.Run("Uint", func(b *testing.B) {
		data := AppendUint(nil, 123456)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Uint()
		}
	})

	b.Run("MapHeader", func(b *testing.B) {
		data := AppendMapHeader(nil, 5)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Len()
		}
	})

	b.Run("Nil", func(b *testing.B) {
		data := AppendNil(nil)
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Type()
		}
	})

	b.Run("String", func(b *testing.B) {
		data := AppendString(nil, "example string")
		iter := NewIterator(nil)
		iter.ResetBytes(data)
		iter.Next()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = iter.Str()
		}
	})

	for format, formatName := range tsFormatStrings {
		b.Run("Timestamp_"+formatName, func(b *testing.B) {
			data := AppendTimestamp(nil, time.Unix(1672531200, 500000000), TsFormat(format))
			iter := NewIterator(nil)
			iter.ResetBytes(data)
			iter.Next()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = iter.Time()
			}
		})
	}
}

func BenchmarkIterator_Skip(b *testing.B) {
	msg := buildComplexMessage()
	iter := NewIterator(bytes.NewReader(msg))

	b.ResetTimer()

	for range b.N {
		iter.ResetBytes(msg)

		for iter.Next() {
			iter.Skip()
		}
	}
}

func BenchmarkIterator_BinReader(b *testing.B) {
	msg := buildComplexMessage()
	iter := NewIterator(bytes.NewReader(msg))

	b.ResetTimer()

	var i int

	for range b.N {
		iter.ResetBytes(msg)
		i = 0

		for iter.Next() {
			_ = iter.BinReader()
			i++
		}
	}

	b.ReportMetric(float64(b.Elapsed())/float64(i)/float64(b.N), "ns/field")
}
