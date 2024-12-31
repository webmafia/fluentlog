package bench

import (
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

var msgpackLengths = [256]byte{
	0xc4: 1,
}

func init() {
	// Initialize the msgpackLengths array
	for i := 0x00; i <= 0x7f; i++ {
		msgpackLengths[i] = 0
	}
	for i := 0xe0; i <= 0xff; i++ {
		msgpackLengths[i] = 0
	}
	for i := 0x80; i <= 0x8f; i++ {
		msgpackLengths[i] = 0
	}
	for i := 0x90; i <= 0x9f; i++ {
		msgpackLengths[i] = 0
	}
	for i := 0xa0; i <= 0xbf; i++ {
		msgpackLengths[i] = 0
	}
	msgpackLengths[0xc0] = 0
	msgpackLengths[0xc2] = 0
	msgpackLengths[0xc3] = 0
	msgpackLengths[0xc4] = 1
	msgpackLengths[0xc5] = 2
	msgpackLengths[0xc6] = 4
	msgpackLengths[0xc7] = 1
	msgpackLengths[0xc8] = 2
	msgpackLengths[0xc9] = 4
	msgpackLengths[0xca] = 4
	msgpackLengths[0xcb] = 8
	msgpackLengths[0xcc] = 1
	msgpackLengths[0xcd] = 2
	msgpackLengths[0xce] = 4
	msgpackLengths[0xcf] = 8
	msgpackLengths[0xd0] = 1
	msgpackLengths[0xd1] = 2
	msgpackLengths[0xd2] = 4
	msgpackLengths[0xd3] = 8
	msgpackLengths[0xd9] = 1
	msgpackLengths[0xda] = 2
	msgpackLengths[0xdb] = 4
	msgpackLengths[0xdc] = 2
	msgpackLengths[0xdd] = 4
	msgpackLengths[0xde] = 2
	msgpackLengths[0xdf] = 4
}

// LengthFromArray returns the length using the precomputed array
func LengthFromArray(b byte) byte {
	return msgpackLengths[b]
}

// LengthFromSwitch returns the length using a switch statement
func LengthFromSwitch(b byte) byte {
	switch b {
	case 0xc0, 0xc2, 0xc3: // nil, false, true
		return 0
	case 0xc4, 0xc7, 0xd9: // bin8, ext8, str8
		return 1
	case 0xc5, 0xc8, 0xda, 0xdc, 0xde: // bin16, ext16, str16, array16, map16
		return 2
	case 0xc6, 0xc9, 0xdb, 0xdd, 0xdf: // bin32, ext32, str32, array32, map32
		return 4
	case 0xca: // float32
		return 4
	case 0xcb: // float64
		return 8
	case 0xcc, 0xd0: // uint8, int8
		return 1
	case 0xcd, 0xd1: // uint16, int16
		return 2
	case 0xce, 0xd2: // uint32, int32
		return 4
	case 0xcf, 0xd3: // uint64, int64
		return 8
	default:
		if b >= 0x00 && b <= 0x7f || b >= 0xe0 && b <= 0xff || b >= 0x80 && b <= 0x9f || b >= 0xa0 && b <= 0xbf {
			return 0
		}
		return 0 // Default to 0 for unknown types
	}
}

func intFromBuf[T int](b []byte) (v T) {
	l := len(b) - 1

	for i := range b {
		v |= T(b[l-i]) << T(8*i)
	}

	return
}

func intFromSwitch[T int](b []byte) T {
	switch len(b) {

	case 1:
		return T(b[0])

	case 2:
		return T(binary.BigEndian.Uint16(b))

	case 4:
		return T(binary.BigEndian.Uint32(b))

	case 8:
		return T(binary.BigEndian.Uint64(b))
	}

	return 0
}

func Benchmark_intDecode(b *testing.B) {
	buf := binary.BigEndian.AppendUint64(make([]byte, 0, 8), 12345678)

	b.Run("intFromSwitch", func(b *testing.B) {
		for range b.N {
			_ = intFromSwitch(buf)
		}
	})

	b.Run("intFromBuf", func(b *testing.B) {
		for range b.N {
			_ = intFromBuf(buf)
		}
	})
}

// BenchmarkLengthFromArray benchmarks the array-based lookup
func BenchmarkLengthFromArray(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = LengthFromArray(0xc4)
	}
}

// BenchmarkLengthFromSwitch benchmarks the switch-based lookup
func BenchmarkLengthFromSwitch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = LengthFromSwitch(0xc4)
	}
}

func BenchmarkClosureInlining(b *testing.B) {
	// fn :=
	c := 1

	b.ResetTimer()

	for i := range b.N {
		_ = doSomething(i, func(a, b int) int {
			return a + b + c
		})
	}
}

func doSomething(i int, fn func(a, b int) int) int {
	return fn(i, 123) + 456
}

func Benchmark_getLength(b *testing.B) {
	b.Run("Fixstr", func(b *testing.B) {
		for range b.N {
			_, _, err := getLength(0xbf, func(l int) ([]byte, error) {
				return nil, nil
			})

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Str", func(b *testing.B) {
		buf := []byte{1, 2, 3, 4}
		b.ResetTimer()

		for range b.N {
			_, _, err := getLength(0xdb, func(l int) ([]byte, error) {
				return buf, nil
			})

			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func getLength(c byte, fn func(l int) ([]byte, error)) (typ types.Type, length int, err error) {
	typ, length, isValueLength := types.Get(c)

	if length > 0 && !isValueLength {
		length, err = getEncodedLength(length, fn)
	}

	return
}

func getEncodedLength(length int, fn func(l int) ([]byte, error)) (int, error) {
	buf, err := fn(length)

	if err != nil {
		return 0, err
	}

	if len(buf) != length {
		return 0, fmt.Errorf("expected %d bytes, got %d bytes", length, len(buf))
	}

	switch length {

	case 1:
		return int(buf[0]), nil

	case 2:
		return int(binary.BigEndian.Uint16(buf)), nil

	case 4:
		return int(binary.BigEndian.Uint32(buf)), nil

	case 8:
		return int(binary.BigEndian.Uint64(buf)), nil

	}

	return 0, errors.New("invalid length")
}
