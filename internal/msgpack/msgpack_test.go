package msgpack

import (
	"bytes"
	"fmt"
	"time"

	"github.com/webmafia/fast/buffer"
)

func Example() {
	var b []byte
	b = AppendArray(b, 3)
	b = AppendString(b, "foo.bar")
	b = AppendTimestamp(b, time.Now())
	b = AppendMap(b, 3)

	b = AppendString(b, "a")
	b = AppendBool(b, true)

	b = AppendString(b, "b")
	b = AppendInt(b, 123)

	// b = AppendString(b, "c")
	// b = AppendFloat64(b, 456.789)

	fmt.Println(len(b), ":", b)

	r := NewReader(bytes.NewReader(b), buffer.NewBuffer(64))

	fmt.Println(r.Read())
	fmt.Println(r.Read())
	fmt.Println(r.Read())
	fmt.Println(r.Read())

	fmt.Println(r.Read())
	fmt.Println(r.Read())

	fmt.Println(r.Read())
	v, _ := r.Read()

	fmt.Println(v.Int())
	// fmt.Println(r.Read())
	// fmt.Println(r.Read())

	// fmt.Println(r.Read())
	// fmt.Println(r.Read())

	// Output: TODO
}
