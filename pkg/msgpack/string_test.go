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

// AppendStringDynamic verifies that AppendStringDynamic produces
// the minimal header for a given string and that the encoded data matches.
func TestAppendStringDynamic(t *testing.T) {
	// Each test case provides an input string and the expected header bytes.
	tests := []struct {
		name   string
		input  string
		header []byte // Expected header prefix.
	}{
		{
			name:   "fixstr short",
			input:  "hello", // length = 5
			header: []byte{0xa0 | 5},
		},
		{
			name:   "fixstr max",
			input:  string(make([]byte, 31)), // length = 31
			header: []byte{0xa0 | 31},
		},
		{
			name:   "str8",
			input:  string(make([]byte, 32)), // length = 32 -> needs str8
			header: []byte{0xd9, 32},
		},
		{
			name:   "str8 upper",
			input:  string(make([]byte, 255)), // length = 255 -> still str8
			header: []byte{0xd9, 255},
		},
		{
			name:   "str16",
			input:  string(make([]byte, 256)), // length = 256 -> needs str16
			header: []byte{0xda, 1, 0},        // 256 = 0x0100
		},
		{
			name:   "str16 max",
			input:  string(make([]byte, 65535)), // length = 65535 -> str16
			header: []byte{0xda, 0xff, 0xff},
		},
		{
			name:   "str32",
			input:  string(make([]byte, 65536)), // length = 65536 -> requires str32
			header: []byte{0xdb, 0, 1, 0, 0},    // 65536 = 0x00010000
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := []byte{}
			s := tc.input
			// fn appends s to the destination.
			fn := func(dst []byte) []byte {
				return append(dst, s...)
			}
			encoded := AppendStringDynamic(dst, fn)
			l := len(s)
			headerLen := len(tc.header)

			// Check that the header is correct.
			if len(encoded) < headerLen {
				t.Fatalf("encoded length %d is less than expected header length %d", len(encoded), headerLen)
			}
			for i := 0; i < headerLen; i++ {
				if encoded[i] != tc.header[i] {
					t.Errorf("header byte %d: got 0x%x, expected 0x%x", i, encoded[i], tc.header[i])
				}
			}

			// Check that the string data follows immediately after the header.
			data := encoded[headerLen:]
			if string(data) != s {
				t.Errorf("data mismatch: got %q, expected %q", string(data), s)
			}

			// Verify the overall length is header length + string length.
			if len(encoded) != headerLen+l {
				t.Errorf("unexpected encoded length: got %d, expected %d", len(encoded), headerLen+l)
			}
		})
	}
}

// BenchmarkAppendStringDynamic benchmarks the performance of AppendStringDynamic.
func BenchmarkAppendStringDynamic(b *testing.B) {
	// We'll use a 100-byte string for this benchmark.
	s := string(make([]byte, 100))
	fn := func(dst []byte) []byte {
		return append(dst, s...)
	}
	var buf []byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = AppendStringDynamic(buf[:0], fn)
	}
}
