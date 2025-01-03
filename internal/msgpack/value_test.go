package msgpack

import (
	"fmt"
	"testing"
)

func ExampleValue_Array() {
	var buf Value

	buf = AppendArray(buf, 3)
	buf = AppendString(buf, "foo")
	buf = AppendString(buf, "bar")
	buf = AppendString(buf, "baz")

	fmt.Println(buf.Len())

	for v := range buf.Array() {
		fmt.Println(v.String())
	}

	// Output: TODO
}

func ExampleValue_Map() {
	var buf Value

	buf = AppendMap(buf, 3)
	buf = AppendString(buf, "foo")
	buf = AppendInt(buf, 123)
	buf = AppendString(buf, "bar")
	buf = AppendInt(buf, 456)
	buf = AppendString(buf, "baz")
	buf = AppendInt(buf, 789)

	fmt.Println(buf.Len())

	for k, v := range buf.Map() {
		fmt.Println(k.String(), v.String())
	}

	// Output: TODO
}

func BenchmarkValueLen(b *testing.B) {
	var v Value
	v = AppendArray(v, 3)

	b.ResetTimer()

	for range b.N {
		_ = v.Len()
	}
}
