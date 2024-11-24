package msgpack

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

func TestReader_ReadArrayHeader(t *testing.T) {
	data := []byte{0x92} // fixarray with 2 elements
	reader := NewReader(bytes.NewReader(data), 1024)

	length, err := reader.ReadArrayHeader()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if length != 2 {
		t.Fatalf("expected length 2, got %d", length)
	}
}

func TestReader_ReadMapHeader(t *testing.T) {
	data := []byte{0x82} // fixmap with 2 key-value pairs
	reader := NewReader(bytes.NewReader(data), 1024)

	length, err := reader.ReadMapHeader()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if length != 2 {
		t.Fatalf("expected length 2, got %d", length)
	}
}

func TestReader_ReadString(t *testing.T) {
	data := []byte{0xa5, 'h', 'e', 'l', 'l', 'o'} // fixstr "hello"
	reader := NewReader(bytes.NewReader(data), 1024)

	str, err := reader.ReadString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if str != "hello" {
		t.Fatalf("expected 'hello', got '%s'", str)
	}
}

func TestReader_ReadInt(t *testing.T) {
	data := []byte{0xd2, 0x00, 0x00, 0x01, 0x2c} // int32 with value 300
	reader := NewReader(bytes.NewReader(data), 1024)

	val, err := reader.ReadInt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 300 {
		t.Fatalf("expected 300, got %d", val)
	}
}

func TestReader_ReadUint(t *testing.T) {
	data := []byte{0xce, 0x00, 0x00, 0x01, 0x2c} // uint32 with value 300
	reader := NewReader(bytes.NewReader(data), 1024)

	val, err := reader.ReadUint()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 300 {
		t.Fatalf("expected 300, got %d", val)
	}
}

func TestReader_ReadBool(t *testing.T) {
	data := []byte{0xc3} // true
	reader := NewReader(bytes.NewReader(data), 1024)

	val, err := reader.ReadBool()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val {
		t.Fatalf("expected true, got false")
	}
}

func TestReader_ReadNil(t *testing.T) {
	data := []byte{0xc0} // nil
	reader := NewReader(bytes.NewReader(data), 1024)

	err := reader.ReadNil()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReader_ReadBinary(t *testing.T) {
	data := []byte{0xc4, 0x03, 0x01, 0x02, 0x03} // bin8 with 3 bytes
	reader := NewReader(bytes.NewReader(data), 1024)

	val, err := reader.ReadBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []byte{0x01, 0x02, 0x03}
	if !bytes.Equal(val, expected) {
		t.Fatalf("expected %v, got %v", expected, val)
	}
}

func TestReader_ReadTimestamp(t *testing.T) {
	data := []byte{0xd6, 0xff, 0x00, 0x00, 0x01, 0x2c} // fixext4 timestamp with 300 seconds
	reader := NewReader(bytes.NewReader(data), 1024)

	val, err := reader.ReadTimestamp()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Unix(300, 0).UTC()
	if !val.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, val)
	}
}

func TestReader_EmptyInput(t *testing.T) {
	reader := NewReader(bytes.NewReader([]byte{}), 1024)

	_, err := reader.ReadArrayHeader()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestReader_Release(t *testing.T) {
	data := []byte{0xa5, 'h', 'e', 'l', 'l', 'o', 0x92, 0x01, 0x02} // fixstr "hello" and fixarray with 2 elements
	reader := NewReader(bytes.NewReader(data), 1024)

	str, err := reader.ReadString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if str != "hello" {
		t.Fatalf("expected 'hello', got '%s'", str)
	}

	reader.Release()

	length, err := reader.ReadArrayHeader()
	if err != nil {
		t.Fatalf("unexpected error after release: %v", err)
	}
	if length != 2 {
		t.Fatalf("expected array length 2, got %d", length)
	}
}
