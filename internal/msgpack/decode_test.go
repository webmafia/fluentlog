package msgpack

import (
	"fmt"
	"testing"
	"time"
)

func errorEqual(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if (a == nil) != (b == nil) {
		return false
	}
	return a.Error() == b.Error()
}

func TestReadArrayHeader(t *testing.T) {
	tests := []struct {
		input     []byte
		offset    int
		length    int
		newOffset int
		err       error
	}{
		// fixarray
		{[]byte{0x90}, 0, 0, 1, nil},
		{[]byte{0x9f}, 0, 15, 1, nil},
		// array16
		{[]byte{0xdc, 0x00, 0x10}, 0, 16, 3, nil},
		{[]byte{0xdc, 0xff, 0xff}, 0, 65535, 3, nil},
		// array32
		{[]byte{0xdd, 0x00, 0x01, 0x00, 0x00}, 0, 65536, 5, nil},
		// Error cases
		{[]byte{}, 0, 0, 0, ErrShortBuffer},
		{[]byte{0xdc}, 0, 0, 1, ErrShortBuffer},                                // incomplete array16
		{[]byte{0xdd}, 0, 0, 1, ErrShortBuffer},                                // incomplete array32
		{[]byte{0x80}, 0, 0, 1, fmt.Errorf("invalid array header byte: 0x80")}, // invalid header
	}

	for _, tt := range tests {
		length, newOffset, err := ReadArrayHeader(tt.input, tt.offset)
		if length != tt.length || newOffset != tt.newOffset || !errorEqual(err, tt.err) {
			t.Errorf("ReadArrayHeader(%x, %d) = (%d, %d, %v); want (%d, %d, %v)",
				tt.input, tt.offset, length, newOffset, err, tt.length, tt.newOffset, tt.err)
		}
	}
}

func TestReadMapHeader(t *testing.T) {
	tests := []struct {
		input     []byte
		offset    int
		length    int
		newOffset int
		err       error
	}{
		// fixmap
		{[]byte{0x80}, 0, 0, 1, nil},
		{[]byte{0x8f}, 0, 15, 1, nil},
		// map16
		{[]byte{0xde, 0x00, 0x10}, 0, 16, 3, nil},
		{[]byte{0xde, 0xff, 0xff}, 0, 65535, 3, nil},
		// map32
		{[]byte{0xdf, 0x00, 0x01, 0x00, 0x00}, 0, 65536, 5, nil},
		// Error cases
		{[]byte{}, 0, 0, 0, ErrShortBuffer},
		{[]byte{0xde}, 0, 0, 1, ErrShortBuffer},                              // incomplete map16
		{[]byte{0xdf}, 0, 0, 1, ErrShortBuffer},                              // incomplete map32
		{[]byte{0x90}, 0, 0, 1, fmt.Errorf("invalid map header byte: 0x90")}, // invalid header
	}

	for _, tt := range tests {
		length, newOffset, err := ReadMapHeader(tt.input, tt.offset)
		if length != tt.length || newOffset != tt.newOffset || !errorEqual(err, tt.err) {
			t.Errorf("ReadMapHeader(%x, %d) = (%d, %d, %v); want (%d, %d, %v)",
				tt.input, tt.offset, length, newOffset, err, tt.length, tt.newOffset, tt.err)
		}
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		input     []byte
		offset    int
		str       string
		newOffset int
		err       error
	}{
		// fixstr
		{append([]byte{0xa0}, []byte("")...), 0, "", 1, nil},
		{append([]byte{0xa5}, []byte("hello")...), 0, "hello", 6, nil},
		// str8
		{append([]byte{0xd9, 0x05}, []byte("hello")...), 0, "hello", 7, nil},
		// str16
		{append([]byte{0xda, 0x00, 0x05}, []byte("hello")...), 0, "hello", 8, nil},
		// str32
		{append([]byte{0xdb, 0x00, 0x00, 0x00, 0x05}, []byte("hello")...), 0, "hello", 10, nil},
		// Error cases
		{[]byte{}, 0, "", 0, ErrShortBuffer},
		{[]byte{0xd9}, 0, "", 1, ErrShortBuffer},                                 // incomplete str8
		{[]byte{0xda}, 0, "", 1, ErrShortBuffer},                                 // incomplete str16
		{[]byte{0xdb}, 0, "", 1, ErrShortBuffer},                                 // incomplete str32
		{[]byte{0xc0}, 0, "", 1, fmt.Errorf("invalid string header byte: 0xc0")}, // invalid header
	}

	for _, tt := range tests {
		str, newOffset, err := ReadString(tt.input, tt.offset)
		if str != tt.str || newOffset != tt.newOffset || !errorEqual(err, tt.err) {
			t.Errorf("ReadString(%x, %d) = (%q, %d, %v); want (%q, %d, %v)",
				tt.input, tt.offset, str, newOffset, err, tt.str, tt.newOffset, tt.err)
		}
	}
}

func TestReadInt(t *testing.T) {
	tests := []struct {
		input     []byte
		offset    int
		value     int64
		newOffset int
		err       error
	}{
		// positive fixint
		{[]byte{0x00}, 0, 0, 1, nil},
		{[]byte{0x7f}, 0, 127, 1, nil},
		// negative fixint
		{[]byte{0xff}, 0, -1, 1, nil},
		{[]byte{0xe0}, 0, -32, 1, nil},
		// int8
		{[]byte{0xd0, 0x80}, 0, int64(int8(-128)), 2, nil},
		// int16
		{[]byte{0xd1, 0x80, 0x00}, 0, int64(int16(-32768)), 3, nil},
		// int32
		{[]byte{0xd2, 0x80, 0x00, 0x00, 0x00}, 0, int64(int32(-2147483648)), 5, nil},
		// int64 incomplete
		{[]byte{0xd3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00}, 0, 0, 1, ErrShortBuffer}, // incomplete
		// uint8
		{[]byte{0xcc, 0x80}, 0, 128, 2, nil},
		// uint16
		{[]byte{0xcd, 0x80, 0x00}, 0, 32768, 3, nil},
		// uint32
		{[]byte{0xce, 0x80, 0x00, 0x00, 0x00}, 0, 2147483648, 5, nil},
		// Error cases
		{[]byte{}, 0, 0, 0, ErrShortBuffer},
		{[]byte{0xd0}, 0, 0, 1, ErrShortBuffer}, // incomplete int8
	}

	for _, tt := range tests {
		value, newOffset, err := ReadInt(tt.input, tt.offset)
		if value != tt.value || newOffset != tt.newOffset || !errorEqual(err, tt.err) {
			t.Errorf("ReadInt(%x, %d) = (%d, %d, %v); want (%d, %d, %v)",
				tt.input, tt.offset, value, newOffset, err, tt.value, tt.newOffset, tt.err)
		}
	}
}

func TestReadBool(t *testing.T) {
	tests := []struct {
		input     []byte
		offset    int
		value     bool
		newOffset int
		err       error
	}{
		{[]byte{0xc2}, 0, false, 1, nil},
		{[]byte{0xc3}, 0, true, 1, nil},
		// Error cases
		{[]byte{}, 0, false, 0, ErrShortBuffer},
		{[]byte{0x00}, 0, false, 1, fmt.Errorf("invalid bool header byte: 0x00")},
	}

	for _, tt := range tests {
		value, newOffset, err := ReadBool(tt.input, tt.offset)
		if value != tt.value || newOffset != tt.newOffset || !errorEqual(err, tt.err) {
			t.Errorf("ReadBool(%x, %d) = (%v, %d, %v); want (%v, %d, %v)",
				tt.input, tt.offset, value, newOffset, err, tt.value, tt.newOffset, tt.err)
		}
	}
}

func TestReadTimestamp(t *testing.T) {
	testTime := time.Unix(1577836800, 0).UTC()
	input := []byte{0xd6, 0xff, 0x5e, 0x0b, 0xe1, 0x00}
	tParsed, newOffset, err := ReadTimestamp(input, 0)
	if err != nil {
		t.Errorf("ReadTimestamp failed: %v", err)
	}
	if !tParsed.Equal(testTime) {
		t.Errorf("ReadTimestamp returned time %v; want %v", tParsed, testTime)
	}
	if newOffset != 6 {
		t.Errorf("ReadTimestamp returned newOffset %d; want %d", newOffset, 6)
	}
}

func Example_decode() {
	var buf []byte

	buf = AppendString(buf, "foobar foobar foobar foobar foobar foobar foobar foobar foobar")

	// fmt.Println(types.Get(buf[0]))
	// fmt.Println(types.GetLength(buf[0]))

	// fmt.Println(ReadString(buf, 0))
	fmt.Println(ReadString(buf, 0))

	// Output: foobar foobar foobar foobar foobar foobar foobar foobar foobar 64 <nil>
}

func BenchmarkString(b *testing.B) {
	var buf []byte

	buf = AppendString(buf, "foobar foobar foobar foobar foobar foobar foobar foobar foobar")

	b.Run("ReadString", func(b *testing.B) {
		for range b.N {
			_, _, _ = ReadString(buf, 0)
		}
	})
}

func ExampleSkip() {
	var buf []byte

	buf = AppendArray(buf, 3)
	buf = AppendString(buf, "foo")
	buf = AppendString(buf, "bar")
	buf = AppendString(buf, "baz")

	fmt.Println(len(buf), string(buf))

	var (
		offset int
		err    error
	)

	fmt.Println(offset)

	if offset, err = Skip(buf, 0); err != nil {
		panic(err)
	}

	fmt.Println(offset)

	// Output: TODO
}
