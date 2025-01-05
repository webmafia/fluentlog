// integer_test.go
package msgpack

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// TestAppendInt tests the AppendInt function for various integer values.
func TestAppendInt(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected []byte
	}{
		// Positive Integers (Encoded as TypeUint)
		{
			name:     "AppendInt Positive FixInt - 0",
			input:    0,
			expected: []byte{0x00},
		},
		{
			name:     "AppendInt Positive FixInt - 127",
			input:    127,
			expected: []byte{0x7f},
		},
		{
			name:     "AppendInt Uint8 - 128",
			input:    128,
			expected: []byte{0xcc, 0x80},
		},
		{
			name:     "AppendInt Uint8 - 255",
			input:    255,
			expected: []byte{0xcc, 0xff},
		},
		{
			name:     "AppendInt Uint16 - 256",
			input:    256,
			expected: []byte{0xcd, 0x01, 0x00},
		},
		{
			name:     "AppendInt Uint16 - 65535",
			input:    65535,
			expected: []byte{0xcd, 0xff, 0xff},
		},
		{
			name:     "AppendInt Uint32 - 65536",
			input:    65536,
			expected: []byte{0xce, 0x00, 0x01, 0x00, 0x00},
		},
		{
			name:     "AppendInt Uint32 - 4294967295",
			input:    4294967295,
			expected: []byte{0xce, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name:     "AppendInt Uint64 - 4294967296",
			input:    4294967296,
			expected: []byte{0xcf, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "AppendInt Uint64 - MaxUint64",
			input:    math.MaxInt64,
			expected: []byte{0xcf, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},

		// Negative Integers (Encoded as TypeInt)
		{
			name:     "AppendInt Negative FixInt - -1",
			input:    -1,
			expected: []byte{0xff},
		},
		{
			name:     "AppendInt Negative FixInt - -32",
			input:    -32,
			expected: []byte{0xe0},
		},
		{
			name:     "AppendInt Int8 - -33",
			input:    -33,
			expected: []byte{0xd0, 0xdf},
		},
		{
			name:     "AppendInt Int8 - -128",
			input:    -128,
			expected: []byte{0xd0, 0x80},
		},
		{
			name:     "AppendInt Int16 - -129",
			input:    -129,
			expected: []byte{0xd1, 0xff, 0x7f},
		},
		{
			name:     "AppendInt Int16 - -32768",
			input:    -32768,
			expected: []byte{0xd1, 0x80, 0x00},
		},
		{
			name:     "AppendInt Int32 - -32769",
			input:    -32769,
			expected: []byte{0xd2, 0xff, 0xff, 0x7f, 0xff},
		},
		{
			name:     "AppendInt Int32 - -2147483648",
			input:    -2147483648,
			expected: []byte{0xd2, 0x80, 0x00, 0x00, 0x00},
		},
		{
			name:     "AppendInt Int64 - -2147483649",
			input:    -2147483649,
			expected: []byte{0xd3, 0xff, 0xff, 0xff, 0xff, 0x7f, 0xff, 0xff, 0xff},
		},
		{
			name:     "AppendInt Int64 - -9223372036854775808",
			input:    math.MinInt64,
			expected: []byte{0xd3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			buf = AppendInt(buf, tt.input)

			if !bytes.Equal(buf, tt.expected) {
				t.Errorf("AppendInt() = %v, expected %v", buf, tt.expected)
			}
		})
	}
}

// TestReadInt tests the ReadInt function for various MessagePack-encoded integers.
func TestReadInt(t *testing.T) {
	tests := []struct {
		name      string
		src       []byte
		offset    int
		want      int64
		wantOff   int
		expectErr error
	}{
		// Reading TypeUint Headers (Positive Integers)
		{
			name:    "ReadInt Positive FixInt - 0",
			src:     []byte{0x00},
			offset:  0,
			want:    0,
			wantOff: 1,
		},
		{
			name:    "ReadInt Positive FixInt - 127",
			src:     []byte{0x7f},
			offset:  0,
			want:    127,
			wantOff: 1,
		},
		{
			name:    "ReadInt Uint8 - 128",
			src:     []byte{0xcc, 0x80},
			offset:  0,
			want:    128,
			wantOff: 2,
		},
		{
			name:    "ReadInt Uint8 - 255",
			src:     []byte{0xcc, 0xff},
			offset:  0,
			want:    255,
			wantOff: 2,
		},
		{
			name:    "ReadInt Uint16 - 256",
			src:     []byte{0xcd, 0x01, 0x00},
			offset:  0,
			want:    256,
			wantOff: 3,
		},
		{
			name:    "ReadInt Uint16 - 65535",
			src:     []byte{0xcd, 0xff, 0xff},
			offset:  0,
			want:    65535,
			wantOff: 3,
		},
		{
			name:    "ReadInt Uint32 - 65536",
			src:     []byte{0xce, 0x00, 0x01, 0x00, 0x00},
			offset:  0,
			want:    65536,
			wantOff: 5,
		},
		{
			name:    "ReadInt Uint32 - 4294967295",
			src:     []byte{0xce, 0xff, 0xff, 0xff, 0xff},
			offset:  0,
			want:    4294967295,
			wantOff: 5,
		},
		{
			name:    "ReadInt Uint64 - 4294967296",
			src:     []byte{0xcf, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			offset:  0,
			want:    4294967296,
			wantOff: 9,
		},
		{
			name:      "ReadInt Uint64 - MaxUint64 Overflows int64",
			src:       []byte{0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: fmt.Errorf("uint64 value %s overflows int64", "18446744073709551615"),
		},
		{
			name:    "ReadInt Uint64 - MaxInt64",
			src:     []byte{0xcf, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			offset:  0,
			want:    math.MaxInt64,
			wantOff: 9,
		},

		// Reading TypeInt Headers (Negative and Positive Integers)
		{
			name:    "ReadInt Negative FixInt - -1",
			src:     []byte{0xff},
			offset:  0,
			want:    -1,
			wantOff: 1,
		},
		{
			name:    "ReadInt Negative FixInt - -32",
			src:     []byte{0xe0},
			offset:  0,
			want:    -32,
			wantOff: 1,
		},
		{
			name:    "ReadInt Int8 - -33",
			src:     []byte{0xd0, 0xdf},
			offset:  0,
			want:    -33,
			wantOff: 2,
		},
		{
			name:    "ReadInt Int8 - -128",
			src:     []byte{0xd0, 0x80},
			offset:  0,
			want:    -128,
			wantOff: 2,
		},
		{
			name:    "ReadInt Int16 - -129",
			src:     []byte{0xd1, 0xff, 0x7f},
			offset:  0,
			want:    -129,
			wantOff: 3,
		},
		{
			name:    "ReadInt Int16 - -32768",
			src:     []byte{0xd1, 0x80, 0x00},
			offset:  0,
			want:    -32768,
			wantOff: 3,
		},
		{
			name:    "ReadInt Int32 - -32769",
			src:     []byte{0xd2, 0xff, 0xff, 0x7f, 0xff},
			offset:  0,
			want:    -32769,
			wantOff: 5,
		},
		{
			name:    "ReadInt Int32 - -2147483648",
			src:     []byte{0xd2, 0x80, 0x00, 0x00, 0x00},
			offset:  0,
			want:    -2147483648,
			wantOff: 5,
		},
		{
			name:    "ReadInt Int64 - -2147483649",
			src:     []byte{0xd3, 0xff, 0xff, 0xff, 0xff, 0x7f, 0xff, 0xff, 0xff},
			offset:  0,
			want:    -2147483649,
			wantOff: 9,
		},
		{
			name:    "ReadInt Int64 - -9223372036854775808",
			src:     []byte{0xd3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			offset:  0,
			want:    math.MinInt64,
			wantOff: 9,
		},
		{
			name:    "ReadInt Int8 - 127",
			src:     []byte{0xd0, 0x7f},
			offset:  0,
			want:    127,
			wantOff: 2,
		},
		{
			name:    "ReadInt Int16 - 32767",
			src:     []byte{0xd1, 0x7f, 0xff},
			offset:  0,
			want:    32767,
			wantOff: 3,
		},
		{
			name:    "ReadInt Int32 - 2147483647",
			src:     []byte{0xd2, 0x7f, 0xff, 0xff, 0xff},
			offset:  0,
			want:    2147483647,
			wantOff: 5,
		},
		{
			name:    "ReadInt Int64 - 9223372036854775807",
			src:     []byte{0xd3, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			offset:  0,
			want:    math.MaxInt64,
			wantOff: 9,
		},

		// Error Cases
		{
			name:      "ReadInt Invalid Header Byte - 0xd4",
			src:       []byte{0xd4, 0x00, 0x00, 0x00},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: fmt.Errorf("invalid int header byte: 0xd4"),
		},
		{
			name:      "ReadInt Short Buffer for Int8",
			src:       []byte{0xd0},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: ErrShortBuffer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOff, err := ReadInt(tt.src, tt.offset)

			// Check for expected error
			if tt.expectErr != nil {
				if err == nil {
					t.Errorf("ReadInt() error = nil, expected %v", tt.expectErr)
					return
				}
				if err.Error() != tt.expectErr.Error() {
					t.Errorf("ReadInt() error = %v, expected %v", err, tt.expectErr)
					return
				}
				// If an error is expected, no need to check further
				return
			} else {
				if err != nil && tt.expectErr == nil {
					t.Errorf("ReadInt() unexpected error = %v", err)
					return
				}
			}

			// Compare values
			if got != tt.want {
				t.Errorf("ReadInt() got = %v, want %v", got, tt.want)
			}

			// Compare newOffset
			if gotOff != tt.wantOff {
				t.Errorf("ReadInt() newOffset = %v, want %v", gotOff, tt.wantOff)
			}
		})
	}
}

// TestAppendUint tests the AppendUint function for various unsigned integer values.
func TestAppendUint(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected []byte
	}{
		{
			name:     "AppendUint Positive FixInt - 0",
			input:    0,
			expected: []byte{0x00},
		},
		{
			name:     "AppendUint Positive FixInt - 127",
			input:    127,
			expected: []byte{0x7f},
		},
		{
			name:     "AppendUint Uint8 - 128",
			input:    128,
			expected: []byte{0xcc, 0x80},
		},
		{
			name:     "AppendUint Uint8 - 255",
			input:    255,
			expected: []byte{0xcc, 0xff},
		},
		{
			name:     "AppendUint Uint16 - 256",
			input:    256,
			expected: []byte{0xcd, 0x01, 0x00},
		},
		{
			name:     "AppendUint Uint16 - 65535",
			input:    65535,
			expected: []byte{0xcd, 0xff, 0xff},
		},
		{
			name:     "AppendUint Uint32 - 65536",
			input:    65536,
			expected: []byte{0xce, 0x00, 0x01, 0x00, 0x00},
		},
		{
			name:     "AppendUint Uint32 - 4294967295",
			input:    4294967295,
			expected: []byte{0xce, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name:     "AppendUint Uint64 - 4294967296",
			input:    4294967296,
			expected: []byte{0xcf, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "AppendUint Uint64 - MaxUint64",
			input:    math.MaxUint64,
			expected: []byte{0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			buf = AppendUint(buf, tt.input)

			if !bytes.Equal(buf, tt.expected) {
				t.Errorf("AppendUint() = %v, expected %v", buf, tt.expected)
			}
		})
	}
}

// TestReadUint tests the ReadUint function for various MessagePack-encoded unsigned integers.
func TestReadUint(t *testing.T) {
	tests := []struct {
		name      string
		src       []byte
		offset    int
		want      uint64
		wantOff   int
		expectErr error
	}{
		// Valid Uint Headers
		{
			name:    "ReadUint Positive FixInt - 0",
			src:     []byte{0x00},
			offset:  0,
			want:    0,
			wantOff: 1,
		},
		{
			name:    "ReadUint Positive FixInt - 127",
			src:     []byte{0x7f},
			offset:  0,
			want:    127,
			wantOff: 1,
		},
		{
			name:    "ReadUint Uint8 - 128",
			src:     []byte{0xcc, 0x80},
			offset:  0,
			want:    128,
			wantOff: 2,
		},
		{
			name:    "ReadUint Uint8 - 255",
			src:     []byte{0xcc, 0xff},
			offset:  0,
			want:    255,
			wantOff: 2,
		},
		{
			name:    "ReadUint Uint16 - 256",
			src:     []byte{0xcd, 0x01, 0x00},
			offset:  0,
			want:    256,
			wantOff: 3,
		},
		{
			name:    "ReadUint Uint16 - 65535",
			src:     []byte{0xcd, 0xff, 0xff},
			offset:  0,
			want:    65535,
			wantOff: 3,
		},
		{
			name:    "ReadUint Uint32 - 65536",
			src:     []byte{0xce, 0x00, 0x01, 0x00, 0x00},
			offset:  0,
			want:    65536,
			wantOff: 5,
		},
		{
			name:    "ReadUint Uint32 - 4294967295",
			src:     []byte{0xce, 0xff, 0xff, 0xff, 0xff},
			offset:  0,
			want:    4294967295,
			wantOff: 5,
		},
		{
			name:    "ReadUint Uint64 - 4294967296",
			src:     []byte{0xcf, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			offset:  0,
			want:    4294967296,
			wantOff: 9,
		},
		{
			name:    "ReadUint Uint64 - MaxUint64",
			src:     []byte{0xcf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			offset:  0,
			want:    math.MaxUint64,
			wantOff: 9,
		},

		// Error Cases
		{
			name:      "ReadUint Invalid Header Byte - 0xd0",
			src:       []byte{0xd0, 0x01},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: fmt.Errorf("invalid uint header byte: 0xd0"),
		},
		{
			name:      "ReadUint Negative FixInt Attempted as Uint",
			src:       []byte{0xe0},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: fmt.Errorf("invalid uint header byte: 0xe0"),
		},
		{
			name:      "ReadUint Short Buffer for Uint8",
			src:       []byte{0xcc},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: ErrShortBuffer,
		},
		{
			name:      "ReadUint Unsupported Uint Length",
			src:       []byte{0xd4, 0x00, 0x00, 0x00},
			offset:    0,
			want:      0,
			wantOff:   0,
			expectErr: fmt.Errorf("invalid uint header byte: 0xd4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOff, err := ReadUint(tt.src, tt.offset)

			// Check for expected error
			if tt.expectErr != nil {
				if err == nil {
					t.Errorf("ReadUint() error = nil, expected %v", tt.expectErr)
					return
				}
				if err.Error() != tt.expectErr.Error() {
					t.Errorf("ReadUint() error = %v, expected %v", err, tt.expectErr)
					return
				}
				// If an error is expected, no need to check further
				return
			} else {
				if err != nil && tt.expectErr == nil {
					t.Errorf("ReadUint() unexpected error = %v", err)
					return
				}
			}

			// Compare values
			if got != tt.want {
				t.Errorf("ReadUint() got = %v, want %v", got, tt.want)
			}

			// Compare newOffset
			if gotOff != tt.wantOff {
				t.Errorf("ReadUint() newOffset = %v, want %v", gotOff, tt.wantOff)
			}
		})
	}
}
