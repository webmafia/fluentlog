package gzip

import (
	"bytes"
	stdgzip "compress/gzip"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/webmafia/fast"
	"github.com/webmafia/fast/bufio"
)

func sampleData() (b []byte, err error) {
	var buf bytes.Buffer

	if err = writeSampleData(&buf); err != nil {
		return
	}

	return buf.Bytes(), nil
}

func writeSampleData(buf *bytes.Buffer) (err error) {
	zw := stdgzip.NewWriter(buf)
	// zw := gzip.NewWriter(buf)

	// Setting the Header fields is optional.
	zw.Name = "a-new-hope.txt"
	zw.Comment = "an epic space opera by George Lucas"
	zw.ModTime = time.Date(1977, time.May, 25, 0, 0, 0, 0, time.UTC)

	if _, err = zw.Write([]byte("A long time ago in a galaxy far, far away...")); err != nil {
		return
	}

	return zw.Close()
}

func ExampleReader() {
	var buf bytes.Buffer

	if err := writeSampleData(&buf); err != nil {
		panic(err)
	}

	br := bufio.NewReader(&buf)
	r, err := NewReader(br)

	if err != nil {
		panic(err)
	}

	data, err := io.ReadAll(r)

	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output: A long time ago in a galaxy far, far away...
}

func ExampleReader_Reset() {
	var buf bytes.Buffer

	if err := writeSampleData(&buf); err != nil {
		panic(err)
	}

	r, err := NewReader(nil)

	if err != nil {
		panic(err)
	}

	if err := r.Reset(nil); err != nil {
		panic(err)
	}

	// Output: TODO
}

func BenchmarkReader_Reset(b *testing.B) {
	buf, err := sampleData()

	if err != nil {
		b.Fatal(err)
	}

	bufReader := bytes.NewReader(buf)
	br := bufio.NewReader(bufReader)
	r, err := NewReader(br)

	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		bufReader.Reset(buf)

		if err := r.Reset(br); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N*len(buf))/(1024*1024)/b.Elapsed().Seconds(), "MB/s")
}

func BenchmarkReader(b *testing.B) {
	buf, err := sampleData()

	if err != nil {
		b.Fatal(err)
	}

	br := bufio.NewReader(nil)
	br.ResetBytes(buf)
	r, err := NewReader(br)

	if err != nil {
		b.Fatal(err)
	}

	var copyBuf [4096]byte

	b.ResetTimer()

	for range b.N {
		br.ResetBytes(buf)

		if err := r.Reset(br); err != nil {
			b.Fatal(err)
		}

		if _, err := io.CopyBuffer(io.Discard, r, copyBuf[:]); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N*len(buf))/(1024*1024)/b.Elapsed().Seconds(), "MB/s")
}

func BenchmarkReader2(b *testing.B) {
	buf, err := sampleData()

	if err != nil {
		b.Fatal(err)
	}

	br := bytes.NewReader(buf)
	r, err := gzip.NewReader(br)

	if err != nil {
		b.Fatal(err)
	}

	var copyBuf [4096]byte

	b.ResetTimer()

	for range b.N {
		br.Reset(buf)

		if err := r.Reset(br); err != nil {
			b.Fatal(err)
		}

		if _, err := io.CopyBuffer(io.Discard, r, copyBuf[:]); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N*len(buf))/(1024*1024)/b.Elapsed().Seconds(), "MB/s")
}

func BenchmarkSkipGzipHeader(b *testing.B) {
	buf, err := sampleData()

	if err != nil {
		b.Fatal(err)
	}

	bufReader := bytes.NewReader(buf)
	b.ResetTimer()

	for range b.N {
		bufReader.Reset(buf)

		if err := SkipGzipHeader(bufReader); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N*len(buf))/(1024*1024)/b.Elapsed().Seconds(), "MB/s")
}

func SkipGzipHeader(r io.Reader) error {
	// Read the basic 10-byte header
	var hdr [10]byte
	if _, err := io.ReadFull(r, fast.NoescapeBytes(hdr[:])); err != nil {
		return err
	}

	// Check the gzip magic numbers
	if hdr[0] != 0x1f || hdr[1] != 0x8b {
		return errors.New("invalid GZIP magic number")
	}

	// Check compression method (must be 8 for DEFLATE)
	if hdr[2] != 8 {
		return errors.New("unsupported GZIP compression method")
	}

	// FLG bits
	flg := hdr[3]

	// If FEXTRA is set, skip the extra field
	if flg&0x04 != 0 {
		var extraLen [2]byte
		if _, err := io.ReadFull(r, fast.NoescapeBytes(extraLen[:])); err != nil {
			return err
		}
		xlen := int(extraLen[0]) | int(extraLen[1])<<8
		if err := skipN(r, xlen); err != nil {
			return err
		}
	}

	// If FNAME is set, skip the filename (null-terminated string)
	if flg&0x08 != 0 {
		if err := skipNullTerminated(r); err != nil {
			return err
		}
	}

	// If FCOMMENT is set, skip the comment (null-terminated string)
	if flg&0x10 != 0 {
		if err := skipNullTerminated(r); err != nil {
			return err
		}
	}

	// If FHCRC is set, skip the 2-byte header CRC
	if flg&0x02 != 0 {
		if err := skipN(r, 2); err != nil {
			return err
		}
	}

	return nil
}

// skipN discards exactly n bytes from r with no heap allocations.
func skipN(r io.Reader, n int) error {
	var buf [256]byte // fixed-size local buffer on the stack
	for n > 0 {
		chunkSize := 256
		if n < chunkSize {
			chunkSize = n
		}
		readBytes, err := io.ReadFull(r, fast.NoescapeBytes(buf[:chunkSize]))
		if err != nil {
			return err
		}
		n -= readBytes
	}
	return nil
}

// skipNullTerminated reads and discards data until it encounters
// a single zero byte. It uses a single-byte stack buffer, so no
// heap allocations occur.
func skipNullTerminated(r io.Reader) error {
	var one [1]byte
	for {
		if _, err := r.Read(fast.NoescapeBytes(one[:])); err != nil {
			return err
		}
		if one[0] == 0 {
			return nil
		}
	}
}
