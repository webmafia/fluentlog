package internal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"testing"
	"time"
)

func writeSampleData(buf *bytes.Buffer) (err error) {
	zw := gzip.NewWriter(buf)

	// Setting the Header fields is optional.
	zw.Name = "a-new-hope.txt"
	zw.Comment = "an epic space opera by George Lucas"
	zw.ModTime = time.Date(1977, time.May, 25, 0, 0, 0, 0, time.UTC)

	if _, err = zw.Write([]byte("A long time ago in a galaxy far, far away...")); err != nil {
		return
	}

	return zw.Close()
}

func ExampleGzipPool() {
	var pool GzipPool
	var buf bytes.Buffer

	if err := writeSampleData(&buf); err != nil {
		panic(err)
	}

	r, err := pool.Acquire(&buf)

	if err != nil {
		panic(err)
	}

	pool.Release(r)

	fmt.Printf("%#v\n", r)

	// Output: TODO
}

func BenchmarkGzipPool(b *testing.B) {
	var pool GzipPool
	var buf bytes.Buffer

	if err := writeSampleData(&buf); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		r, err := pool.Acquire(&buf)

		if err != nil {
			b.Fatal(err)
		}

		if err = pool.Release(r); err != nil {
			b.Fatal(err)
		}
	}
}
