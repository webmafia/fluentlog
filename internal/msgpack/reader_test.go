package msgpack

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func ExampleReader() {
	var b []byte

	b = AppendArray(b, 3)
	b = AppendString(b, "foo.bar")
	b = AppendTimestamp(b, time.Now())
	// b = AppendMap(b, 3)

	// b = AppendString(b, "a")
	// b = AppendBool(b, true)

	// b = AppendString(b, "b")
	// b = AppendInt(b, 123)

	// b = AppendString(b, "c")
	// b = AppendFloat64(b, 456.789)

	r := NewReader(bytes.NewReader(b), make([]byte, 4096))

	fmt.Println(r.PeekType())
	fmt.Println(r.ReadArrayHeader())

	fmt.Println(r.PeekType())
	fmt.Println(r.ReadString())

	fmt.Println(r.PeekType())
	fmt.Println(r.ReadTimestamp())

	fmt.Println(r.PeekType())
	fmt.Println(r.ReadMapHeader())

	// Output: TODO
}

func TestReader_ReadRaw(t *testing.T) {
	// MessagePack-encoded data for an array [1, "hello", [true, false]]
	data := []byte{
		0x93,                               // Array of length 3
		0x01,                               // Integer 1
		0xa5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, // String "hello"
		0x92, // Array of length 2
		0xc3, // True
		0xc2, // False
	}

	buffer := make([]byte, 1024)
	reader := NewReader(bytes.NewReader(data), buffer)

	rawBytes, err := reader.ReadRaw()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that rawBytes matches the original data
	if !bytes.Equal(rawBytes, data) {
		t.Fatalf("expected %x, got %x", data, rawBytes)
	}
}
