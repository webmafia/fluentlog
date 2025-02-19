package msgpack

import (
	"bytes"
	"testing"
	"time"
)

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
			_ = iter.Reader()
			i++
		}
	}

	b.ReportMetric(float64(b.Elapsed())/float64(i)/float64(b.N), "ns/field")
}
