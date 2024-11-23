package msgpack

import (
	"bytes"
	"math"
	"testing"
	"time"
)

func TestAppendArray(t *testing.T) {
	tests := []struct {
		n    int
		want []byte
	}{
		{0, []byte{0x90}},
		{15, []byte{0x9f}},
		{16, []byte{0xdc, 0x00, 0x10}},
		{65535, []byte{0xdc, 0xff, 0xff}},
		{65536, []byte{0xdd, 0x00, 0x01, 0x00, 0x00}},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendArray(dst, tt.n)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendArray(%d) = %x; want %x", tt.n, got, tt.want)
		}
	}
}

func TestAppendMap(t *testing.T) {
	tests := []struct {
		n    int
		want []byte
	}{
		{0, []byte{0x80}},
		{15, []byte{0x8f}},
		{16, []byte{0xde, 0x00, 0x10}},
		{65535, []byte{0xde, 0xff, 0xff}},
		{65536, []byte{0xdf, 0x00, 0x01, 0x00, 0x00}},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendMap(dst, tt.n)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendMap(%d) = %x; want %x", tt.n, got, tt.want)
		}
	}
}

func TestAppendString(t *testing.T) {
	tests := []struct {
		s    string
		want []byte
	}{
		{"", []byte{0xa0}},
		{"hello", append([]byte{0xa5}, []byte("hello")...)},
		{string(make([]byte, 31)), append([]byte{0xbf}, make([]byte, 31)...)},
		{string(make([]byte, 32)), append([]byte{0xd9, 0x20}, make([]byte, 32)...)},
		{string(make([]byte, 255)), append([]byte{0xd9, 0xff}, make([]byte, 255)...)},
		{string(make([]byte, 256)), append([]byte{0xda, 0x01, 0x00}, make([]byte, 256)...)},
		{string(make([]byte, 65535)), append([]byte{0xda, 0xff, 0xff}, make([]byte, 65535)...)},
		{string(make([]byte, 65536)), append([]byte{0xdb, 0x00, 0x01, 0x00, 0x00}, make([]byte, 65536)...)},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendString(dst, tt.s)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendString(%d bytes) = %x; want %x", len(tt.s), got, tt.want)
		}
	}
}

func TestAppendInt(t *testing.T) {
	tests := []struct {
		i    int64
		want []byte
	}{
		{0, []byte{0x00}},
		{127, []byte{0x7f}},
		{128, []byte{0xcc, 0x80}},                          // uint8
		{255, []byte{0xcc, 0xff}},                          // uint8
		{256, []byte{0xcd, 0x01, 0x00}},                    // uint16
		{32767, []byte{0xcd, 0x7f, 0xff}},                  // uint16
		{32768, []byte{0xcd, 0x80, 0x00}},                  // uint16
		{65535, []byte{0xcd, 0xff, 0xff}},                  // uint16
		{65536, []byte{0xce, 0x00, 0x01, 0x00, 0x00}},      // uint32
		{2147483647, []byte{0xce, 0x7f, 0xff, 0xff, 0xff}}, // uint32
		{2147483648, []byte{0xce, 0x80, 0x00, 0x00, 0x00}}, // uint32
		{-1, []byte{0xff}},
		{-32, []byte{0xe0}},
		{-33, []byte{0xd0, 0xdf}},
		{-128, []byte{0xd0, 0x80}},
		{-129, []byte{0xd1, 0xff, 0x7f}},
		{-32768, []byte{0xd1, 0x80, 0x00}},
		{-32769, []byte{0xd2, 0xff, 0xff, 0x7f, 0xff}},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendInt(dst, tt.i)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendInt(%d) = %x; want %x", tt.i, got, tt.want)
		}
	}
}

func TestAppendUint(t *testing.T) {
	tests := []struct {
		u    uint64
		want []byte
	}{
		// Positive FixInt
		{0, []byte{0x00}},
		{127, []byte{0x7f}},
		// uint8
		{128, []byte{0xcc, 0x80}},
		{255, []byte{0xcc, 0xff}},
		// uint16
		{256, []byte{0xcd, 0x01, 0x00}},
		{65535, []byte{0xcd, 0xff, 0xff}},
		// uint32
		{65536, []byte{0xce, 0x00, 0x01, 0x00, 0x00}},
		{4294967295, []byte{0xce, 0xff, 0xff, 0xff, 0xff}},
		// uint64
		{4294967296, []byte{0xcf, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}},
		{18446744073709551615, []byte{0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, // Max uint64
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendUint(dst, tt.u)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendUint(%v) = %x; want %x", tt.u, got, tt.want)
		}
	}
}

func TestAppendNil(t *testing.T) {
	dst := []byte{}
	got := AppendNil(dst)
	want := []byte{0xc0}
	if !bytes.Equal(got, want) {
		t.Errorf("AppendNil() = %x; want %x", got, want)
	}
}

func TestAppendBool(t *testing.T) {
	tests := []struct {
		b    bool
		want []byte
	}{
		{true, []byte{0xc3}},
		{false, []byte{0xc2}},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendBool(dst, tt.b)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendBool(%v) = %x; want %x", tt.b, got, tt.want)
		}
	}
}

func TestAppendBinary(t *testing.T) {
	tests := []struct {
		data []byte
		want []byte
	}{
		{[]byte{}, []byte{0xc4, 0x00}},
		{make([]byte, 255), append([]byte{0xc4, 0xff}, make([]byte, 255)...)},
		{make([]byte, 256), append([]byte{0xc5, 0x01, 0x00}, make([]byte, 256)...)},
		{make([]byte, 65535), append([]byte{0xc5, 0xff, 0xff}, make([]byte, 65535)...)},
		{make([]byte, 65536), append([]byte{0xc6, 0x00, 0x01, 0x00, 0x00}, make([]byte, 65536)...)},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendBinary(dst, tt.data)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendBinary(%d bytes) = %x; want %x", len(tt.data), got, tt.want)
		}
	}
}

func TestAppendFloat32(t *testing.T) {
	tests := []struct {
		f    float32
		want []byte
	}{
		{0.0, []byte{0xca, 0x00, 0x00, 0x00, 0x00}},
		{math.Pi, []byte{0xca, 0x40, 0x49, 0x0f, 0xdb}},
		{-math.Pi, []byte{0xca, 0xc0, 0x49, 0x0f, 0xdb}},
		{math.MaxFloat32, []byte{0xca, 0x7f, 0x7f, 0xff, 0xff}},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendFloat32(dst, tt.f)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendFloat32(%v) = %x; want %x", tt.f, got, tt.want)
		}
	}
}

func TestAppendFloat64(t *testing.T) {
	tests := []struct {
		f    float64
		want []byte
	}{
		{0.0, []byte{0xcb, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{math.Pi, []byte{0xcb, 0x40, 0x09, 0x21, 0xfb, 0x54, 0x44, 0x2d, 0x18}},
		{-math.Pi, []byte{0xcb, 0xc0, 0x09, 0x21, 0xfb, 0x54, 0x44, 0x2d, 0x18}},
		{math.MaxFloat64, []byte{0xcb, 0x7f, 0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}
	for _, tt := range tests {
		dst := []byte{}
		got := AppendFloat64(dst, tt.f)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendFloat64(%v) = %x; want %x", tt.f, got, tt.want)
		}
	}
}

func TestAppendTimestamp(t *testing.T) {
	testTime := time.Unix(1577836800, 0).UTC() // 2020-01-01 00:00:00 UTC
	dst := []byte{}
	got := AppendTimestamp(dst, testTime)
	want := []byte{0xd6, 0xff, 0x5e, 0x0b, 0xe1, 0x00} // fixext4, -1, seconds

	if !bytes.Equal(got, want) {
		t.Errorf("AppendTimestamp(%v) = %x; want %x", testTime, got, want)
	}
}

func TestAppendFunctionsCombined(t *testing.T) {
	dst := []byte{}
	dst = AppendArray(dst, 4) // [tag, time, record, option]

	// Append tag
	tag := "myapp.access" // 12 characters
	dst = AppendString(dst, tag)

	// Append timestamp
	now := time.Unix(1577836800, 0).UTC() // Fixed time
	dst = AppendTimestamp(dst, now)

	// Append record map with one key-value pair
	dst = AppendMap(dst, 1)
	dst = AppendString(dst, "message")
	dst = AppendString(dst, "hello world")

	// Append empty option map
	dst = AppendMap(dst, 0)

	// Expected bytes
	want := []byte{
		0x94,                                                       // array of 4 elements
		0xac,                                                       // fixstr with length 12
		'm', 'y', 'a', 'p', 'p', '.', 'a', 'c', 'c', 'e', 's', 's', // "myapp.access"
		0xd6, 0xff, // fixext4, -1
		0x5e, 0x0b, 0xe1, 0x00, // seconds: 1577836800
		0x81,                                    // map with one key-value pair
		0xa7, 'm', 'e', 's', 's', 'a', 'g', 'e', // key: "message"
		0xab, 'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', // value: "hello world"
		0x80, // empty map
	}
	if !bytes.Equal(dst, want) {
		t.Errorf("Combined append functions result = %x; want %x", dst, want)
	}
}
