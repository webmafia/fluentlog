package bench

import "testing"

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
