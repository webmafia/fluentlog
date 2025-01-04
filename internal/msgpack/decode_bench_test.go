package msgpack

import (
	"testing"
)

func BenchmarkReadInt(b *testing.B) {
	src := []byte{0xd1, 0xff, 0x85} // Example int16 value: -123
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadInt(src, offset)
	}
}

func BenchmarkReadUint(b *testing.B) {
	src := []byte{0xcd, 0x00, 0xff} // Example uint16 value: 255
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadUint(src, offset)
	}
}

func BenchmarkReadString(b *testing.B) {
	src := []byte{0xd9, 0x05, 'h', 'e', 'l', 'l', 'o'} // Example string: "hello"
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadString(src, offset)
	}
}

func BenchmarkReadBool(b *testing.B) {
	src := []byte{0xc3} // Example boolean: true
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadBool(src, offset)
	}
}

func BenchmarkReadNil(b *testing.B) {
	src := []byte{0xc0} // Example nil value
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _ = ReadNil(src, offset)
	}
}

func BenchmarkReadBinary(b *testing.B) {
	src := []byte{0xc4, 0x03, 0x01, 0x02, 0x03} // Example binary: [1, 2, 3]
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadBinary(src, offset)
	}
}

func BenchmarkReadFloat32(b *testing.B) {
	src := []byte{0xca, 0x41, 0x20, 0x00, 0x00} // Example float32: 10.0
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadFloat32(src, offset)
	}
}

func BenchmarkReadFloat64(b *testing.B) {
	src := []byte{0xcb, 0x40, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // Example float64: 10.0
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadFloat64(src, offset)
	}
}

func BenchmarkReadTimestamp(b *testing.B) {
	src := []byte{0xd7, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // Example timestamp: 0
	offset := 0
	for i := 0; i < b.N; i++ {
		_, _, _ = ReadEventTime(src, offset)
	}
}
