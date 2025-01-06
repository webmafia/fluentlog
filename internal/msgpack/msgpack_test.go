package msgpack

import (
	"bytes"
	"fmt"
	"time"

	"github.com/webmafia/fast/buffer"
)

func Example() {
	var b []byte
	b = AppendArrayHeader(b, 3)
	b = AppendString(b, "foo.bar")
	b = AppendTimestamp(b, time.Now())
	// b = AppendMapHeader(b, 3)

	// b = AppendString(b, "a")
	// b = AppendBool(b, true)

	// b = AppendString(b, "b")
	// b = AppendInt(b, -123)

	// b = AppendString(b, "c")
	// b = AppendFloat64(b, 456.789)

	fmt.Println(len(b), ":", b)

	r := NewReader(bytes.NewReader(b), buffer.NewBuffer(64), 4096)

	fmt.Println(r.Read()) // AppendArrayHeader(b, 3)
	fmt.Println(r.Read()) // AppendString(b, "foo.bar")
	v, _ := r.Read()
	fmt.Println(v) // AppendEventTime(b, time.Now())

	// fmt.Println(r.Read())

	// fmt.Println(r.Read())
	// fmt.Println(r.Read())

	// fmt.Println(r.Read())
	// v, _ := r.Read()

	// fmt.Println(v)
	// fmt.Println(r.Read())
	// fmt.Println(r.Read())

	// fmt.Println(r.Read())
	// fmt.Println(r.Read())

	// Output: TODO
}
