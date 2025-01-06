package msgpack

import (
	"bytes"
	"testing"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func TestReader(t *testing.T) {
	tests := []struct {
		input          []byte
		expectedType   types.Type
		expectedSubval int
		expectedErr    bool
		description    string
	}{
		// Simple Types
		{[]byte{0xc0}, types.Nil, 0, false, "Nil type"},
		{[]byte{0xc3}, types.Bool, 0, false, "Bool type (true)"},
		{[]byte{0xca, 0x40, 0x49, 0x0f, 0xdb}, types.Float, 0, false, "Float32 type"},

		// Fixed-length integer types
		{[]byte{0xcc, 0xff}, types.Uint, 0, false, "Uint8 type"},
		{[]byte{0xd1, 0x7f, 0xff}, types.Int, 0, false, "Int16 type"},

		// String Types
		{[]byte{0xd9, 0x05, 'h', 'e', 'l', 'l', 'o'}, types.Str, 0, false, "Str8 type with 'hello'"},
		{[]byte{0xdb, 0x00, 0x00, 0x00, 0x06, 'w', 'o', 'r', 'l', 'd', '!'}, types.Str, 0, false, "Str32 type with 'world!'"},

		// Fixed Compound Types
		{[]byte{0x80}, types.Map, 0, false, "FixMap: Empty map"},
		{[]byte{0x85}, types.Map, 5, false, "FixMap: Map with 5 key-value pairs"},
		{[]byte{0x90}, types.Array, 0, false, "FixArray: Empty array"},
		{[]byte{0x9f}, types.Array, 15, false, "FixArray: Array with 15 elements"},

		// Variable-Length Arrays
		{[]byte{0xdc, 0x00, 0x03}, types.Array, 3, false, "Array16 with 3 elements"},
		{[]byte{0xdd, 0x00, 0x00, 0x00, 0x05}, types.Array, 5, false, "Array32 with 5 elements"},
		{[]byte{0xdc, 0x00, 0x00}, types.Array, 0, false, "Array16 with 0 elements (valid header)"},

		// Variable-Length Maps
		{[]byte{0xde, 0x00, 0x02}, types.Map, 2, false, "Map16 with 2 key-value pairs"},
		{[]byte{0xdf, 0x00, 0x00, 0x00, 0x04}, types.Map, 4, false, "Map32 with 4 key-value pairs"},
		{[]byte{0xde, 0x00, 0x00}, types.Map, 0, false, "Map16 with 0 key-value pairs (valid header)"},

		// Longer Byte Slices
		{[]byte{0xc2, 0xcc, 0x01, 0xca, 0x40, 0x49, 0x0f, 0xdb}, types.Bool, 0, false, "Bool type followed by Uint8 and Float32 (only Bool read)"},
		{[]byte{0x91, 0xcc, 0x05}, types.Array, 1, false, "Array with 1 element (Uint8: 5)"},
		{[]byte{0xde, 0x00, 0x01, 0xcc, 0x02, 0xd0, 0x03}, types.Map, 1, false, "Map with 1 key-value pair (Uint8: 2 -> Int8: 3, only header read)"},

		// Error cases
		{[]byte{0xcc}, types.Nil, 0, true, "Truncated Uint8"},
		{[]byte{0xdc}, types.Nil, 0, true, "Truncated Array16 header"},
		{[]byte{0xdf, 0x00, 0x00}, types.Nil, 0, true, "Truncated Map32 header"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			r2 := NewReader(r, buffer.NewBuffer(64), 4096)

			v, err := r2.Read()

			if (err != nil) != tt.expectedErr {
				t.Errorf("Unexpected error: got %v, want error=%v", err, tt.expectedErr)
			}

			if v.Type() != tt.expectedType {
				t.Errorf("Unexpected type: got %v, want %v", v.Type(), tt.expectedType)
			}

			if v.Type() == types.Array || v.Type() == types.Map {
				if n := v.Len(); n != tt.expectedSubval {
					t.Errorf("Unexpected subvalue count: got %d, want %d", n, tt.expectedSubval)
				}
			}

			if !tt.expectedErr && !bytes.Equal(v, tt.input[:len(v)]) {
				t.Errorf("Unexpected output bytes: got %v, want %v", v, tt.input[:len(v)])
			}
		})
	}
}

func BenchmarkReader(b *testing.B) {
	benchmarks := []struct {
		input       []byte
		description string
	}{
		{[]byte{0xc0}, "Nil type"},
		{[]byte{0xca, 0x40, 0x49, 0x0f, 0xdb}, "Float32 type"},
		{[]byte{0xcc, 0xff}, "Uint8 type"},
		{[]byte{0xd9, 0x05, 'h', 'e', 'l', 'l', 'o'}, "Str8 type with 'hello'"},
		{[]byte{0xde, 0x00, 0x02}, "Map16 with 2 key-value pairs"},
		{[]byte{0xdc, 0x00, 0x03}, "Array16 with 3 elements"},
		{[]byte{0xdf, 0x00, 0x00, 0x00, 0x04}, "Map32 with 4 key-value pairs"},
	}

	b.Run("Baseline", func(b *testing.B) {
		r := bytes.NewBuffer(nil)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			r.Reset()
			r.Write([]byte{0xdf, 0x00, 0x00, 0x00, 0x04})
		}
	})

	for _, bm := range benchmarks {
		b.Run(bm.description, func(b *testing.B) {
			r := bytes.NewBuffer(nil)
			r2 := NewReader(r, buffer.NewBuffer(64), 4096)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r.Reset()
				r.Write(bm.input)
				r2.Reset()
				_, _ = r2.Read()
			}
		})
	}
}

func TestRelease(t *testing.T) {
	tests := []struct {
		name        string
		initialData []byte
		rn          int
		releaseN    int
		expectedBuf []byte
		expectedN   int
	}{
		{
			name:        "Release with valid n",
			initialData: []byte{'A', 'B', 'C', 'D', 'E', 'F'},
			rn:          3,
			releaseN:    2,
			expectedBuf: []byte{'A', 'B', 'D', 'E', 'F'},
			expectedN:   5,
		},
		{
			name:        "Release everything",
			initialData: []byte{'A', 'B', 'C'},
			rn:          3,
			releaseN:    0,
			expectedBuf: []byte{},
			expectedN:   0,
		},
		{
			name:        "Release with n beyond r.n",
			initialData: []byte{'A', 'B', 'C'},
			rn:          2,
			releaseN:    3,
			expectedBuf: []byte{'A', 'B', 'C'},
			expectedN:   2,
		},
		{
			name:        "Release with negative n",
			initialData: []byte{'A', 'B', 'C'},
			rn:          2,
			releaseN:    -1,
			expectedBuf: []byte{'A', 'B', 'C'},
			expectedN:   2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := buffer.NewBuffer(64)
			buf.B = append(buf.B, test.initialData...)

			r := Reader{
				b: buf,
				n: test.rn,
			}

			r.Release(test.releaseN)

			if !bytes.Equal(r.b.B, test.expectedBuf) {
				t.Errorf("buffer mismatch: got %v, want %v", r.b.B, test.expectedBuf)
			}

			if r.n != test.expectedN {
				t.Errorf("position mismatch: got %d, want %d", r.n, test.expectedN)
			}
		})
	}
}
