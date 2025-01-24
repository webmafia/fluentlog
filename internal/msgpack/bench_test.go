package msgpack

import (
	"bytes"
	"testing"
	"time"

	"github.com/webmafia/fast/buffer"
)

func BenchmarkAppend(b *testing.B) {
	b.Run("ArrayHeader", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendArrayHeader(buf[:0], 10)
		}
	})

	b.Run("Binary", func(b *testing.B) {
		var buf []byte
		data := []byte("example binary data")
		for i := 0; i < b.N; i++ {
			buf = AppendBinary(buf[:0], data)
		}
	})

	b.Run("Bool", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendBool(buf[:0], true)
		}
	})

	b.Run("Float", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendFloat(buf[:0], 3.14159)
		}
	})

	b.Run("Int", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendInt(buf[:0], -123456)
		}
	})

	b.Run("Uint", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendUint(buf[:0], 123456)
		}
	})

	b.Run("MapHeader", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendMapHeader(buf[:0], 5)
		}
	})

	b.Run("Nil", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendNil(buf[:0])
		}
	})

	b.Run("String", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendString(buf[:0], "example string")
		}
	})

	b.Run("AnyString", func(b *testing.B) {
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf = AppendAny(buf[:0], "example string")
		}
	})

	for format, formatName := range tsFormatStrings {
		b.Run("Timestamp_"+formatName, func(b *testing.B) {
			var buf []byte
			t := time.Unix(1672531200, 500000000)
			for i := 0; i < b.N; i++ {
				buf = AppendTimestamp(buf[:0], t, TsFormat(format))
			}
		})
	}
}

func BenchmarkRead(b *testing.B) {
	b.Run("ArrayHeader", func(b *testing.B) {
		data := AppendArrayHeader(nil, 10)
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadArrayHeader(data, 0)
		}
	})

	b.Run("Binary", func(b *testing.B) {
		data := AppendBinary(nil, []byte("example binary data"))
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadBinary(data, 0)
		}
	})

	b.Run("Bool", func(b *testing.B) {
		data := AppendBool(nil, true)
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadBool(data, 0)
		}
	})

	b.Run("Float", func(b *testing.B) {
		data := AppendFloat(nil, 3.14159)
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadFloat(data, 0)
		}
	})

	b.Run("Int", func(b *testing.B) {
		data := AppendInt(nil, -123456)
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadInt(data, 0)
		}
	})

	b.Run("Uint", func(b *testing.B) {
		data := AppendUint(nil, 123456)
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadUint(data, 0)
		}
	})

	b.Run("MapHeader", func(b *testing.B) {
		data := AppendMapHeader(nil, 5)
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadMapHeader(data, 0)
		}
	})

	b.Run("Nil", func(b *testing.B) {
		data := AppendNil(nil)
		for i := 0; i < b.N; i++ {
			_, _ = ReadNil(data, 0)
		}
	})

	b.Run("String", func(b *testing.B) {
		data := AppendString(nil, "example string")
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadString(data, 0)
		}
	})

	b.Run("StringCopy", func(b *testing.B) {
		data := AppendString(nil, "example string")
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadStringCopy(data, 0)
		}
	})

	for format, formatName := range tsFormatStrings {
		b.Run("Timestamp_"+formatName, func(b *testing.B) {
			data := AppendTimestamp(nil, time.Unix(1672531200, 500000000), TsFormat(format))
			for i := 0; i < b.N; i++ {
				_, _, _ = ReadTimestamp(data, 0)
			}
		})
	}
}

func BenchmarkReadIterator(b *testing.B) {
	b.Run("ArrayHeader", func(b *testing.B) {
		data := AppendArrayHeader(nil, 10) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
		}
	})

	b.Run("Binary", func(b *testing.B) {
		data := AppendBinary(nil, []byte("example binary data")) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
			iter.ReadBinary()
		}
	})

	b.Run("Bool", func(b *testing.B) {
		data := AppendBool(nil, true) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
			iter.ReadBool()
		}
	})

	b.Run("Float", func(b *testing.B) {
		data := AppendFloat(nil, 3.14159) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
			iter.ReadFloat()
		}
	})

	b.Run("Int", func(b *testing.B) {
		data := AppendInt(nil, -123456) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
			iter.ReadInt()
		}
	})

	b.Run("Uint", func(b *testing.B) {
		data := AppendUint(nil, 123456) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
			iter.ReadUint()
		}
	})

	b.Run("MapHeader", func(b *testing.B) {
		data := AppendMapHeader(nil, 5) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
		}
	})

	b.Run("Nil", func(b *testing.B) {
		data := AppendNil(nil) // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
		}
	})

	b.Run("String", func(b *testing.B) {
		data := AppendString(nil, "example string") // Generate test data
		reader := bytes.NewReader(data)
		buf := buffer.NewBuffer(128)
		iter := NewIterator(reader, buf, len(data)) // Define iterator once
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader.Reset(data) // Reset the reader
			iter.Reset(reader) // Reset the iterator
			iter.Next()
			iter.ReadString()
		}
	})

	for format, formatName := range tsFormatStrings {
		b.Run("Timestamp_"+formatName, func(b *testing.B) {
			data := AppendTimestamp(nil, time.Unix(1672531200, 500000000), TsFormat(format)) // Generate test data
			reader := bytes.NewReader(data)
			buf := buffer.NewBuffer(128)
			iter := NewIterator(reader, buf, len(data)) // Define iterator once
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				reader.Reset(data) // Reset the reader
				iter.Reset(reader) // Reset the iterator
				iter.Next()
				iter.ReadTime()
			}
		})
	}
}
