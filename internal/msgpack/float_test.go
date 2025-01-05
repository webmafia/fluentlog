package msgpack

import (
	"math"
	"testing"
)

func TestAppendFloat(t *testing.T) {
	tests := []struct {
		desc   string
		input  float64
		expect []byte
	}{
		{"encode float32", 1.5, []byte{0xca, 0x3f, 0xc0, 0x00, 0x00}},
		{"encode float64", 1.1, []byte{0xcb, 0x3f, 0xf1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a}},
		{"encode large float64", math.Pi, []byte{0xcb, 0x40, 0x09, 0x21, 0xfb, 0x54, 0x44, 0x2d, 0x18}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			dst := []byte{}
			result := AppendFloat(dst, tt.input)
			if len(result) != len(tt.expect) || !equalBytes(result, tt.expect) {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestReadFloat(t *testing.T) {
	tests := []struct {
		desc      string
		input     []byte
		offset    int
		expect    float64
		newOffset int
		expectErr bool
	}{
		{"read float32", []byte{0xca, 0x3f, 0xc0, 0x00, 0x00}, 0, 1.5, 5, false},
		{"read float64", []byte{0xcb, 0x3f, 0xf1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a}, 0, 1.1, 9, false},
		{"read large float64", []byte{0xcb, 0x40, 0x09, 0x21, 0xfb, 0x54, 0x44, 0x2d, 0x18}, 0, math.Pi, 9, false},
		{"invalid header byte", []byte{0xcc, 0x00}, 0, 0, 0, true},
		{"short buffer for float32", []byte{0xca, 0x3f, 0xc0}, 0, 0, 0, true},
		{"short buffer for float64", []byte{0xcb, 0x40, 0x09, 0x21, 0xfb}, 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result, newOffset, err := ReadFloat(tt.input, tt.offset)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected an error but got none")
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

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
