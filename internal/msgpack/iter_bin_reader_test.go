package msgpack

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func Example_binReader() {
	iter := NewIterator(nil, 32)
	data := make([]byte, 95)

	for i := range data {
		data[i] = byte(i + 1)
	}

	var buf []byte
	buf = AppendBinary(buf, data)
	buf = AppendString(buf, "foobar")
	buf = AppendString(buf, "baz")
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
			data:     AppendBinary(nil, []byte("hello world")),
			bufSize:  512, // Larger than the data size
			expected: []byte("hello world"),
		},
		{
			name: "Partially Buffered",
			data: func() []byte {
				// Create a binary payload larger than the buffer size
				return AppendBinary(nil, []byte("this is a partially buffered test"))
			}(),
			bufSize:  16, // Force partial buffering
			expected: []byte("this is a partially buffered test"),
		},
		{
			name: "Large Binary Payload",
			data: func() []byte {
				// Create a large binary payload (10 KB)
				largePayload := bytes.Repeat([]byte("A"), 10*1024)
				return AppendBinary(nil, largePayload)
			}(),
			bufSize:  4096, // Smaller than the total payload
			expected: bytes.Repeat([]byte("A"), 10*1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)

			// Create an iterator with a restricted buffer size
			iter := NewIterator(reader, tt.bufSize)
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

func TestBinReader_ReadByte(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte // Binary payload to test
		bufSize  int    // Max buffer size for the iterator
		expected []byte // Expected binary data
	}{
		{
			name:     "Small Payload",
			data:     AppendBinary(nil, []byte("hello")),
			bufSize:  10, // More than enough buffer size
			expected: []byte("hello"),
		},
		{
			name: "Partial Buffer Fill",
			data: func() []byte {
				// Create a binary payload that forces multiple refills
				return AppendBinary(nil, []byte("this is a test"))
			}(),
			bufSize:  5, // Force small buffer refills
			expected: []byte("this is a test"),
		},
		{
			name: "Large Binary Data",
			data: func() []byte {
				// Large binary payload (8 KB)
				largePayload := bytes.Repeat([]byte("X"), 8*1024)
				return AppendBinary(nil, largePayload)
			}(),
			bufSize:  512, // Much smaller than total payload
			expected: bytes.Repeat([]byte("X"), 8*1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)

			// Create an iterator with a restricted buffer size
			iter := NewIterator(reader, tt.bufSize)
			if !iter.Next() {
				t.Fatalf("Failed to move to the next token in %s", tt.name)
			}

			// Use BinReader to read the binary payload byte-by-byte
			binReader := iter.BinReader()
			if binReader == nil {
				t.Fatalf("BinReader returned nil in %s", tt.name)
			}

			var result []byte
			for {
				b, err := binReader.(io.ByteReader).ReadByte()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Unexpected error while reading byte in %s: %v", tt.name, err)
				}
				result = append(result, b)
			}

			// Verify the binary payload
			if !bytes.Equal(tt.expected, result) {
				t.Errorf("Binary payload does not match in %s. Expected %v, got %v", tt.name, tt.expected, result)
			}
		})
	}
}

func BenchmarkBinReader_ReadByte(b *testing.B) {
	tests := []struct {
		name    string
		payload []byte // Raw binary data
		bufSize int    // Max buffer size for the iterator
	}{
		{
			name:    "Small Payload",
			payload: []byte("hello world"),
			bufSize: 16, // More than enough buffer space
		},
		{
			name:    "Medium Payload",
			payload: bytes.Repeat([]byte("X"), 1024), // 1 KB payload
			bufSize: 64,                              // Forces multiple refills
		},
		{
			name:    "Large Payload",
			payload: bytes.Repeat([]byte("Y"), 10*1024), // 10 KB payload
			bufSize: 512,                                // Much smaller than total payload
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Encode the payload as MessagePack binary for each run
			data := AppendBinary(nil, tt.payload)
			// Reinitialize a fresh reader for every iteration
			reader := bytes.NewReader(data)
			iter := NewIterator(reader, tt.bufSize)

			// Reset benchmark timer
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				reader.Reset(data)
				iter.Reset(reader)

				// Ensure we correctly move to the next token
				if !iter.Next() {
					b.Fatalf("Failed to move to the next token in %s", tt.name)
				}

				binReader := iter.BinReader()

				// Read the binary data byte-by-byte
				for {
					_, err := binReader.ReadByte()
					if err == io.EOF {
						break
					}
					if err != nil {
						b.Fatalf("Unexpected error while reading byte in %s: %v", tt.name, err)
					}
				}
			}

			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(data)), "ns/byte")
		})
	}
}

func BenchmarkBufioReader_ReadByte(b *testing.B) {
	tests := []struct {
		name    string
		payload []byte // Raw binary data
		bufSize int    // Max buffer size for the iterator
	}{
		{
			name:    "Small Payload",
			payload: []byte("hello world"),
			bufSize: 16, // More than enough buffer space
		},
		{
			name:    "Medium Payload",
			payload: bytes.Repeat([]byte("X"), 1024), // 1 KB payload
			bufSize: 64,                              // Forces multiple refills
		},
		{
			name:    "Large Payload",
			payload: bytes.Repeat([]byte("Y"), 10*1024), // 10 KB payload
			bufSize: 512,                                // Much smaller than total payload
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			reader := bytes.NewReader(tt.payload)
			buf := bufio.NewReader(reader)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				reader.Reset(tt.payload)
				buf.Reset(reader)

				// Read the binary data byte-by-byte
				for {
					_, err := buf.ReadByte()
					if err == io.EOF {
						break
					}
					if err != nil {
						b.Fatalf("Unexpected error while reading byte in %s: %v", tt.name, err)
					}
				}
			}

			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(tt.payload)), "ns/byte")
		})
	}
}

// TestBinReader_SeekByte_Suite thoroughly tests SeekByte with various edge cases,
// including partial reads, multiple refills, and small maxBufSize constraints.
func TestBinReader_SeekByte_Suite(t *testing.T) {

	t.Run("SeekByte on empty BIN data", func(t *testing.T) {
		// 0xC4 with length=0 => empty bin
		raw := []byte{0xC4, 0x00}

		iter := NewIterator(nil)
		iter.ResetBytes(raw)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}

		if got := iter.Type(); got != types.Bin {
			t.Fatalf("Expected bin, got %v", got)
		}

		br := iter.BinReader()
		if err := br.SeekByte('x'); err != io.EOF {
			t.Fatalf("Expected io.EOF seeking in empty data, got %v", err)
		}
	})

	t.Run("SeekByte at beginning of BIN", func(t *testing.T) {
		// 0xC4 0x05 => BIN8 length=5, data: "ABCDE"
		raw := []byte{0xC4, 0x05, 'A', 'B', 'C', 'D', 'E'}

		iter := NewIterator(nil)
		iter.ResetBytes(raw)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		br := iter.BinReader()

		// 'A' is the first byte
		if err := br.SeekByte('A'); err != nil {
			t.Fatalf("SeekByte('A') unexpected error: %v", err)
		}

		// Next read should be 'A'
		ch, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte after SeekByte('A') error: %v", err)
		}
		if ch != 'A' {
			t.Fatalf("Expected 'A', got '%c'", ch)
		}
	})

	t.Run("SeekByte at end of BIN", func(t *testing.T) {
		// 6 bytes: "Hello!" => 0xC4 0x06 + "Hello!"
		raw := append([]byte{0xC4, 0x06}, []byte("Hello!")...)

		iter := NewIterator(nil)
		iter.ResetBytes(raw)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		br := iter.BinReader()

		// '!' is the last byte
		if err := br.SeekByte('!'); err != nil {
			t.Fatalf("SeekByte('!') error: %v", err)
		}

		// Next read should be '!'
		ch, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte after SeekByte('!') error: %v", err)
		}
		if ch != '!' {
			t.Fatalf("Expected '!', got '%c'", ch)
		}

		// Further read should be EOF
		ch, err = br.ReadByte()
		if err != io.EOF {
			t.Fatalf("Expected EOF after last byte, got %v (ch=%c)", err, ch)
		}
	})

	t.Run("SeekByte with multiple occurrences (first occurrence)", func(t *testing.T) {
		// "Hello, world!"
		// The letter 'l' occurs multiple times
		raw := append([]byte{0xC4, 0x0D}, []byte("Hello, world!")...)

		iter := NewIterator(nil)
		iter.ResetBytes(raw)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		br := iter.BinReader()

		if err := br.SeekByte('l'); err != nil {
			t.Fatalf("SeekByte('l') unexpected error: %v", err)
		}

		// The first occurrence of 'l' is the 3rd character in "Hello" ("H=0,e=1,l=2,l=3,...")
		ch, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte() error after seeking 'l': %v", err)
		}
		if ch != 'l' {
			t.Fatalf("Expected 'l', got '%c'", ch)
		}

		// The next read should be the next 'l'
		ch, err = br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte() error after reading first 'l': %v", err)
		}
		if ch != 'l' {
			t.Fatalf("Expected the next 'l', got '%c'", ch)
		}
	})

	t.Run("SeekByte for missing byte -> EOF", func(t *testing.T) {
		raw := append([]byte{0xC4, 0x0D}, []byte("Hello, world!")...)

		iter := NewIterator(nil)
		iter.ResetBytes(raw)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		br := iter.BinReader()

		// 'Z' is not in "Hello, world!"
		if err := br.SeekByte('Z'); err != io.EOF {
			t.Fatalf("Expected io.EOF seeking 'Z', got %v", err)
		}
		// Confirm subsequent reads are also EOF
		if _, err := br.ReadByte(); err != io.EOF {
			t.Fatalf("After unsuccessful SeekByte, expected EOF, got %v", err)
		}
	})

	t.Run("SeekByte after partial reading", func(t *testing.T) {
		raw := append([]byte{0xC4, 0x0D}, []byte("Hello, world!")...)

		iter := NewIterator(nil)
		iter.ResetBytes(raw)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		br := iter.BinReader()

		// Partially read first 7 bytes ("Hello, ")
		buf := make([]byte, 7)
		n, err := br.Read(buf)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
		if n != 7 || string(buf) != "Hello, " {
			t.Fatalf("Unexpected read. got=%q, n=%d", buf, n)
		}

		// Now seek the 'w' in the remaining "world!"
		if err := br.SeekByte('w'); err != nil {
			t.Fatalf("SeekByte('w') error: %v", err)
		}

		// Next read should be 'w'
		ch, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte after SeekByte('w') error: %v", err)
		}
		if ch != 'w' {
			t.Fatalf("Expected 'w', got '%c'", ch)
		}
	})

	t.Run("SeekByte in large data requiring multiple buffer refills", func(t *testing.T) {
		// We'll create a large data chunk so that internal calls to fill() happen.
		// We'll insert 'Z' near the very end to test SeekByte.
		chunk := strings.Repeat("Hello, world!", 500) + "Z"
		data := []byte(chunk)

		buf := buildBinToken(data)

		// We'll force multiple small reads from the underlying source by limiting chunk size.
		limitedReader := &chunkedReader{
			data:   buf,
			chunks: 128, // read at most 128 bytes at a time
		}

		iter := NewIterator(limitedReader)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		br := iter.BinReader()

		// Seek the 'Z' near the end
		if err := br.SeekByte('Z'); err != nil {
			t.Fatalf("SeekByte('Z') in large data error: %v", err)
		}

		// Confirm next byte is indeed 'Z'
		ch, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte error after SeekByte('Z'): %v", err)
		}
		if ch != 'Z' {
			t.Fatalf("Expected 'Z', got '%c'", ch)
		}

		// Any further read should be EOF
		ch, err = br.ReadByte()
		if err != io.EOF {
			t.Fatalf("Expected EOF after last byte, got '%c', err=%v", ch, err)
		}
	})

	t.Run("SeekByte with small maxBufSize but data fits (multiple expansions)", func(t *testing.T) {
		// We test a scenario where data is larger than the default buffer but still
		// smaller than maxBufSize, forcing multiple expansions but NOT reaching the max.
		chunk := "Begin" + strings.Repeat("ABC", 50) + "Z" + "End"
		data := []byte(chunk) // ~ 154 bytes

		raw := buildBinToken(data)
		// We'll let the underlying reader supply everything at once here.
		// The critical part is that the iterator's maxBufSize is small.
		rd := bytes.NewReader(raw)

		// Create Iterator with small maxBufSize.
		// It's bigger than 154, but smaller than the length we might need if we re-fill multiple times.
		iter := NewIterator(rd, 100)
		if !iter.Next() {
			t.Fatalf("Iterator.Next() failed. err: %v", iter.Error())
		}
		if iter.Type() != types.Bin {
			t.Fatalf("Expected Bin type, got %v", iter.Type())
		}
		br := iter.BinReader()

		// We expect to find 'Z' after reading a bunch of 'ABC's
		if err := br.SeekByte('Z'); err != nil {
			t.Fatalf("SeekByte('Z') error: %v", err)
		}

		// Confirm next read is 'Z'
		ch, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte after SeekByte('Z') error: %v", err)
		}
		if ch != 'Z' {
			t.Fatalf("Expected 'Z', got '%c'", ch)
		}

		// Then we expect "End" after that
		buf := make([]byte, 3)
		n, err := br.Read(buf)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
		if string(buf[:n]) != "End" {
			t.Fatalf("Expected 'End', got '%s'", buf[:n])
		}

		// Confirm EOF now
		_, err = br.ReadByte()
		if err != io.EOF {
			t.Fatalf("Expected EOF, got %v", err)
		}
	})

	t.Run("SeekByte with data exceeding maxBufSize -> expected error", func(t *testing.T) {
		// We'll produce data bigger than we can ever buffer,
		// so we expect an internal ErrReachedMaxBufferSize or wrap thereof.
		chunk := strings.Repeat("Hello, world!", 1000) // ~13k bytes

		// Insert 'Z' so that we might try to seek it, but we won't get that far.
		chunk = chunk + "Z"

		raw := buildBinToken([]byte(chunk))

		rd := bytes.NewReader(raw)
		// Set a maxBufSize that's smaller than the total data size
		iter := NewIterator(rd, 512)
		if !iter.Next() {
			// If we fail on the Next() itself, let's see if it's the buffer-size error or something else
			if !errors.Is(iter.Error(), ErrReachedMaxBufferSize) {
				t.Fatalf("Expected ErrReachedMaxBufferSize, got %v", iter.Error())
			}
			// If it's the expected error, we pass
			return
		}

		// If Next() didn't error, try to read and see if we get the error later
		br := iter.BinReader()
		err := br.SeekByte('Z')
		if err == nil {
			t.Fatal("Expected an error or EOF due to exceeding maxBufSize, but got nil")
		}

		// We can check if it’s the special error or a wrapped version
		if !errors.Is(err, ErrReachedMaxBufferSize) && err != io.ErrUnexpectedEOF {
			t.Fatalf("Expected ErrReachedMaxBufferSize or an EOF, got: %v", err)
		}
	})
}

// ------------------------------------------------------------------------------
// chunkedReader is a helper io.Reader that returns data in small chunks.
// This helps ensure multiple refills occur inside the Iterator.
type chunkedReader struct {
	data   []byte
	offset int
	chunks int
}

func (r *chunkedReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	if len(p) > r.chunks {
		p = p[:r.chunks]
	}
	remaining := len(r.data) - r.offset
	if len(p) > remaining {
		p = p[:remaining]
	}
	copy(p, r.data[r.offset:r.offset+len(p)])
	r.offset += len(p)
	return len(p), nil
}

// buildBinToken constructs a MessagePack "Bin" token. Automatically chooses BIN8,
// BIN16, or BIN32 based on length.
func buildBinToken(data []byte) []byte {
	size := len(data)
	var buf bytes.Buffer
	switch {
	case size <= 0xFF:
		// BIN8
		buf.WriteByte(0xC4)
		buf.WriteByte(byte(size))
	case size <= 0xFFFF:
		// BIN16
		buf.WriteByte(0xC5)
		buf.WriteByte(byte(size >> 8))
		buf.WriteByte(byte(size))
	default:
		// BIN32
		buf.WriteByte(0xC6)
		buf.WriteByte(byte(size >> 24))
		buf.WriteByte(byte(size >> 16))
		buf.WriteByte(byte(size >> 8))
		buf.WriteByte(byte(size))
	}
	buf.Write(data)
	return buf.Bytes()
}
