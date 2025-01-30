package msgpack

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"time"
)

func sampleData() (b []byte, err error) {
	var buf bytes.Buffer

	if err = writeSampleData(&buf); err != nil {
		return
	}

	return buf.Bytes(), nil
}

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

func ExampleReader() {
	data, err := sampleData()

	if err != nil {
		panic(err)
	}

	fmt.Println(data)
	data = append(data, data...)
	fmt.Println(data)

	data = AppendBinary(nil, data)
	r := bytes.NewReader(data)
	iter := NewIterator(r)

	if !iter.Next() {
		panic("failed next")
	}

	res, err := io.ReadAll(iter.GzipReader())

	if err != nil {
		panic(err)
	}

	fmt.Println(string(res))

	// Output: A long time ago in a galaxy far, far away...
}
