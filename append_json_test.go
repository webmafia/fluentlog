package fluentlog

import (
	"fmt"
	"testing"
)

type myStruct struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func Example_appendJSON() {
	var buf []byte

	buf = appendJSON(buf, myStruct{
		Foo: "hello",
		Bar: "world",
	})

	fmt.Println(buf[:5])
	fmt.Println(string(buf[5:]))

	// Output:
	//
	// [198 0 0 0 29]
	// {"foo":"hello","bar":"world"}
}

func Benchmark_appendJSON(b *testing.B) {
	var buf []byte
	v := myStruct{
		Foo: "hello",
		Bar: "world",
	}

	for b.Loop() {
		buf = appendJSON(buf[:0], v)
	}
}
