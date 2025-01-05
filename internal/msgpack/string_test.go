package msgpack

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type mockTextAppender struct {
	data string
}

func (m *mockTextAppender) AppendText(b []byte) ([]byte, error) {
	return append(b, m.data...), nil
}

func TestAppendString(t *testing.T) {
	tests := []struct {
		desc   string
		input  string
		expect []byte
	}{
		{"short string", "abc", []byte{0xa3, 'a', 'b', 'c'}},
		{"medium string", strings.Repeat("a", 255), append([]byte{0xd9, 0xff}, bytes.Repeat([]byte{'a'}, 255)...)},
		{"long string", strings.Repeat("a", 65535), append([]byte{0xda, 0xff, 0xff}, bytes.Repeat([]byte{'a'}, 65535)...)},
		{"very long string", strings.Repeat("a", 70000), append([]byte{0xdb, 0x00, 0x01, 0x11, 0x70}, bytes.Repeat([]byte{'a'}, 70000)...)},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			dst := []byte{}
			result := AppendString(dst, tt.input)
			if !bytes.Equal(tt.expect, result) {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		desc      string
		input     []byte
		offset    int
		expect    string
		newOffset int
		expectErr error
	}{
		{"valid short string", []byte{0xa3, 'a', 'b', 'c'}, 0, "abc", 4, nil},
		{"valid medium string", append([]byte{0xd9, 0xff}, bytes.Repeat([]byte{'a'}, 255)...), 0, strings.Repeat("a", 255), 257, nil},
		{"short buffer", []byte{0xa3, 'a'}, 0, "", 0, ErrShortBuffer},
		{"wrong type", []byte{0x90, 'a', 'b', 'c'}, 0, "", 0, ErrInvalidHeaderByte},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result, newOffset, err := ReadString(tt.input, tt.offset)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("expected error %v, got %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expect {
					t.Errorf("expected %v, got %v", tt.expect, result)
				}
				if newOffset != tt.newOffset {
					t.Errorf("expected new offset %v, got %v", tt.newOffset, newOffset)
				}
			}
		})
	}
}

func TestAppendTextAppender(t *testing.T) {
	tests := []struct {
		desc   string
		input  *mockTextAppender
		expect []byte
	}{
		{"append simple string", &mockTextAppender{data: "hello"}, []byte{0xdb, 0, 0, 0, 5, 'h', 'e', 'l', 'l', 'o'}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			dst := []byte{}
			result := AppendTextAppender(dst, tt.input)
			if !bytes.Equal(tt.expect, result) {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestAppendStringUnknownLength(t *testing.T) {
	dst := []byte{}
	result := AppendStringUnknownLength(dst, func(dst []byte) []byte {
		return append(dst, "dynamic-length"...)
	})

	length := len("dynamic-length")
	expect := append([]byte{0xdb, byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}, []byte("dynamic-length")...)
	if !bytes.Equal(expect, result) {
		t.Errorf("expected %v, got %v", expect, result)
	}
}
