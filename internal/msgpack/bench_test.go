package msgpack

import (
	"testing"
	"time"
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

	b.Run("EventTime", func(b *testing.B) {
		var buf []byte
		t := time.Unix(1672531200, 500000000)
		for i := 0; i < b.N; i++ {
			buf = AppendEventTime(buf[:0], t)
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

	b.Run("EventTime", func(b *testing.B) {
		data := AppendEventTime(nil, time.Unix(1672531200, 500000000))
		for i := 0; i < b.N; i++ {
			_, _, _ = ReadEventTime(data, 0)
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
}
