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
	b = AppendTimestamp(b, time.Now(), TsFluentd)
	b = AppendMapHeader(b, 3)

	b = AppendString(b, "a")
	b = AppendBool(b, true)

	b = AppendString(b, "b")
	b = AppendInt(b, -123)

	b = AppendString(b, "c")
	b = AppendFloat(b, 456.789)

	fmt.Println(len(b), ":", b)

	r := NewReader(bytes.NewReader(b), buffer.NewBuffer(64), 4096)

	fmt.Println(r.Read()) // AppendArrayHeader(b, 3)
	fmt.Println(r.Read()) // AppendString(b, "foo.bar")
	fmt.Println(r.Read()) // AppendEventTime(b, time.Now())
	fmt.Println(r.Read()) // AppendMapHeader(b, 3)

	fmt.Println(r.Read())
	fmt.Println(r.Read())

	fmt.Println(r.Read())
	fmt.Println(r.Read())

	fmt.Println(r.Read())
	fmt.Println(r.Read())

	// Output: TODO
}
