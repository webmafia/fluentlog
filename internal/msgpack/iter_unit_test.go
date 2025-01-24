package msgpack

import (
	"bytes"
	"testing"
	"time"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func TestIterator_Next(t *testing.T) {
	var data []byte
	data = AppendArrayHeader(data[:0], 3) // Array header with 3 elements
	data = AppendInt(data, 1)
	data = AppendInt(data, 2)
	data = AppendInt(data, 3)

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	typ, length := iter.Next()
	if typ != types.Array {
		t.Errorf("expected type Array, got %v", typ)
	}
	if length != 3 {
		t.Errorf("expected length 3, got %d", length)
	}
}

func TestIterator_ReadBinary(t *testing.T) {
	var data []byte
	data = AppendBinary(data[:0], []byte("foo")) // Binary data with content "foo"

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the binary data
	result := iter.ReadBinary()
	expected := []byte{'f', 'o', 'o'}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestIterator_ReadString(t *testing.T) {
	var data []byte
	data = AppendString(data[:0], "bar") // String "bar"

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the string data
	result := iter.ReadString()
	expected := "bar"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestIterator_ReadInt(t *testing.T) {
	var data []byte
	data = AppendInt(data[:0], -123456) // Negative integer

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the int data
	result := iter.ReadInt()
	expected := -123456
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

func TestIterator_ReadUint(t *testing.T) {
	var data []byte
	data = AppendUint(data[:0], 123456) // Unsigned integer

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the uint data
	result := iter.ReadUint()
	expected := uint(123456)
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

func TestIterator_ReadFloat(t *testing.T) {
	var data []byte
	data = AppendFloat(data[:0], 3.14159) // Float 3.14159

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the float data
	result := iter.ReadFloat()
	expected := 3.14159
	tolerance := 0.000001

	if diff := result - expected; diff < -tolerance || diff > tolerance {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

func TestIterator_ReadTime(t *testing.T) {
	var data []byte
	expected := time.Unix(1672531200, 500000000) // Example timestamp
	data = AppendTimestamp(data[:0], expected, TsFormat(0))

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the timestamp
	result := iter.ReadTime()
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestIterator_Skip(t *testing.T) {
	var data []byte
	data = AppendArrayHeader(data[:0], 3) // Array with 3 elements
	data = AppendInt(data, 1)
	data = AppendInt(data, 2)
	data = AppendInt(data, 3)

	buf := buffer.NewBuffer(128)
	iter := NewIterator(bytes.NewReader(data), buf, len(data))

	iter.Next() // Move to the array
	skipped := iter.Skip()
	if !skipped {
		t.Error("expected Skip to return true")
	}
	if iter.Total() != len(data) {
		t.Errorf("expected total read bytes %d, got %d", len(data), iter.Total())
	}
}
