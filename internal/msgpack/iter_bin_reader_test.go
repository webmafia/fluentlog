package msgpack_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func Example_binReader() {
	iter := msgpack.NewIterator(nil, 32)
	data := make([]byte, 95)

	for i := range data {
		data[i] = byte(i + 1)
	}

	var buf []byte
	buf = msgpack.AppendBinary(buf, data)
	buf = msgpack.AppendString(buf, "foobar")
	buf = msgpack.AppendString(buf, "baz")
	iter.ResetBytes(buf)

	for iter.Next() {
		fmt.Println(iter.Type(), iter.Len())

		switch iter.Type() {
		case types.Bin:
			var p [10]byte
			r := iter.BinReader()

			for {
				n, err := r.Read(p[:])

				fmt.Println("read", n, "bytes:", p[:n])

				if err != nil {
					fmt.Println("io.Reader error:", err)
					break
				}
			}

		case types.Str:
			fmt.Println("read string:", iter.Str())
		default:
			fmt.Println("unhandled type")
		}

		if err := iter.Error(); err != nil {
			fmt.Println("error:", err)
		}

		// iter.Release(true)
		fmt.Println("---")
	}

	// fmt.Println(iter.Error())

	// Output:
	//
	// bin 95
	// read 10 bytes: [1 2 3 4 5 6 7 8 9 10]
	// read 10 bytes: [11 12 13 14 15 16 17 18 19 20]
	// read 10 bytes: [21 22 23 24 25 26 27 28 29 30]
	// read 10 bytes: [31 32 33 34 35 36 37 38 39 40]
	// read 10 bytes: [41 42 43 44 45 46 47 48 49 50]
	// read 10 bytes: [51 52 53 54 55 56 57 58 59 60]
	// read 10 bytes: [61 62 63 64 65 66 67 68 69 70]
	// read 10 bytes: [71 72 73 74 75 76 77 78 79 80]
	// read 10 bytes: [81 82 83 84 85 86 87 88 89 90]
	// read 5 bytes: [91 92 93 94 95]
	// io.Reader error: EOF
	// ---
	// str 6
	// read string: foobar
	// ---
	// str 3
	// read string: baz
	// ---
}

func TestBinReader(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte // Binary payload to test
		bufSize  int    // Max buffer size for the iterator
		expected []byte // Expected binary data
	}{
		{
			name:     "Fully Buffered",
			data:     msgpack.AppendBinary(nil, []byte("hello world")),
			bufSize:  512, // Larger than the data size
			expected: []byte("hello world"),
		},
		{
			name: "Partially Buffered",
			data: func() []byte {
				// Create a binary payload larger than the buffer size
				return msgpack.AppendBinary(nil, []byte("this is a partially buffered test"))
			}(),
			bufSize:  16, // Force partial buffering
			expected: []byte("this is a partially buffered test"),
		},
		{
			name: "Large Binary Payload",
			data: func() []byte {
				// Create a large binary payload (10 KB)
				largePayload := bytes.Repeat([]byte("A"), 10*1024)
				return msgpack.AppendBinary(nil, largePayload)
			}(),
			bufSize:  4096, // Smaller than the total payload
			expected: bytes.Repeat([]byte("A"), 10*1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)

			// Create an iterator with a restricted buffer size
			iter := msgpack.NewIterator(reader, tt.bufSize)
			if !iter.Next() {
				t.Fatalf("Failed to move to the next token in %s", tt.name)
			}

			// Use BinReader to read the binary payload
			binReader := iter.BinReader()
			if binReader == nil {
				t.Fatalf("BinReader returned nil in %s", tt.name)
			}

			// Read the binary data from the reader in chunks
			buf := make([]byte, 1024) // Read in chunks of 1 KB
			var result []byte
			for {
				n, err := binReader.Read(buf)
				if n > 0 {
					result = append(result, buf[:n]...)
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Unexpected error while reading binary data in %s: %v", tt.name, err)
				}
			}

			// Verify the binary payload
			if !bytes.Equal(tt.expected, result) {
				t.Errorf("Binary payload does not match in %s. Expected %v, got %v", tt.name, tt.expected, result)
			}
		})
	}
}
