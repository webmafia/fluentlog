package msgpack

import (
	"strconv"
	"testing"
)

func BenchmarkIntFromBuf(b *testing.B) {
	buf1 := []byte{0x01}
	buf2 := []byte{0x01, 0x02}
	buf4 := []byte{0x01, 0x02, 0x03, 0x04}
	buf8 := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	b.Run("1 byte", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = intFromBuf[int8](buf1)
		}
	})

	b.Run("2 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = intFromBuf[int16](buf2)
		}
	})

	b.Run("4 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = intFromBuf[int32](buf4)
		}
	})

	b.Run("8 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = intFromBuf[int64](buf8)
		}
	})
}

func BenchmarkUintFromBuf(b *testing.B) {
	buf1 := []byte{0x01}
	buf2 := []byte{0x01, 0x02}
	buf4 := []byte{0x01, 0x02, 0x03, 0x04}
	buf8 := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	b.Run("1 byte", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = uintFromBuf[uint8](buf1)
		}
	})

	b.Run("2 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = uintFromBuf[uint16](buf2)
		}
	})

	b.Run("4 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = uintFromBuf[uint32](buf4)
		}
	})

	b.Run("8 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = uintFromBuf[uint64](buf8)
		}
	})
}

func BenchmarkFloatFromBuf(b *testing.B) {
	buf4 := []byte{0x3f, 0x80, 0x00, 0x00}                         // 1.0 as float32
	buf8 := []byte{0x3f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // 1.0 as float64

	b.Run("4 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = floatFromBuf[float32](buf4)
		}
	})

	b.Run("8 bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = floatFromBuf[float64](buf8)
		}
	})
}

func BenchmarkRoundPow(b *testing.B) {
	values := []int{1, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095}

	for _, v := range values {
		b.Run(strconv.Itoa(v), func(b *testing.B) {
			for range b.N {
				_ = roundPow(v)
			}
		})
	}

}
