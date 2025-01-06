package msgpack

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

func ExampleReadTimestamp() {
	var buf []byte

	t := time.Date(2025, 01, 01, 1, 2, 3, 0, time.UTC)
	buf = AppendTimestamp(buf, t, TsForwardEventTime)

	fmt.Println(buf)

	dst, _, err := ReadTimestamp(buf, 0)

	if err != nil {
		panic(err)
	}

	fmt.Println(t)
	fmt.Println(dst.UTC())

	// Output: TODO
}

func TestAppendEventTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected []byte
	}{
		{
			"EventTime Example",
			time.Unix(1672531200, 500000000),
			[]byte{0xd7, 0x00, 0x63, 0x68, 0x89, 0x80, 0x1d, 0xc9, 0xc3, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendTimestamp(nil, tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestAppendEventTimeShort(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected []byte
	}{
		{
			"Short EventTime Example",
			time.Unix(1672531200, 0),
			AppendInt(nil, 1672531200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendTimestamp(nil, tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %x, got %x", tt.expected, result)
			}
		})
	}
}

func TestReadEventTime(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		offset         int
		expectedTime   time.Time
		expectedOffset int
		expectedErr    error
	}{
		{
			"Valid EventTime",
			[]byte{0xd7, 0x00, 0x63, 0x68, 0x89, 0x80, 0x1d, 0xc9, 0xc3, 0x00},
			0,
			time.Unix(1672531200, 500000000),
			10,
			nil,
		},
		{
			"Valid Short Timestamp",
			AppendInt(nil, 1672531200),
			0,
			time.Unix(1672531200, 0),
			5,
			nil,
		},
		{
			"Invalid Header",
			[]byte{0xd4, 0x00},
			0,
			time.Time{},
			0,
			ErrInvalidHeaderByte,
		},
		{
			"Empty Input",
			[]byte{},
			0,
			time.Time{},
			0,
			io.ErrUnexpectedEOF,
		},
		{
			"Offset Out of Bounds",
			[]byte{0xd7, 0x00, 0x63, 0x68, 0x89, 0x80},
			10,
			time.Time{},
			0,
			io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, newOffset, err := ReadTimestamp(tt.input, tt.offset)

			// Check for expected error
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			// Verify decoded time and offset
			if !result.Equal(tt.expectedTime) {
				t.Errorf("expected time %v, got %v", tt.expectedTime, result)
			}
			if newOffset != tt.expectedOffset {
				t.Errorf("expected newOffset %d, got %d", tt.expectedOffset, newOffset)
			}

			// Ensure no additional data was decoded
			if newOffset < len(tt.input) && tt.input[newOffset] != 0 {
				t.Errorf("unexpected data decoded beyond newOffset")
			}
		})
	}
}
