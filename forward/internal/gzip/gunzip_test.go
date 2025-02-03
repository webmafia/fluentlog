package gzip

import (
	"bytes"
	stdgzip "compress/gzip"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
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

func Example_sampleData() {
	data, _ := sampleData()
	fmt.Println(len(data), data)

	// Output:
	//
	// 117 [31 139 8 24 128 227 232 13 0 255 97 45 110 101 119 45 104 111 112 101 46 116 120 116 0 97 110 32 101 112 105 99 32 115 112 97 99 101 32 111 112 101 114 97 32 98 121 32 71 101 111 114 103 101 32 76 117 99 97 115 0 114 84 200 201 207 75 87 40 201 204 77 85 72 76 207 87 200 204 83 72 84 72 79 204 73 172 168 84 72 75 44 210 1 17 10 137 229 137 149 122 122 122 128 0 0 0 255 255 16 138 163 239 44 0 0 0]
}

func ExampleReader() {
	var buf bytes.Buffer

	if err := writeSampleData(&buf); err != nil {
		panic(err)
	}

	if err := writeSampleData(&buf); err != nil {
		panic(err)
	}

	// br := bufio.NewReader(&buf)
	br := bufio.NewReader(&buf).LimitReader(buf.Len())
	r, err := NewReader(br)

	if err != nil {
		panic(err)
	}

	data, err := io.ReadAll(r)

	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// A long time ago in a galaxy far, far away...A long time ago in a galaxy far, far away...
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
