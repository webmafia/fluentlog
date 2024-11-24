package fluentlog

import (
	"fmt"
	"time"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

func ExampleValue() {
	var b []byte
	b = msgpack.AppendArray(b, 3)
	b = msgpack.AppendString(b, "foo.bar")
	b = msgpack.AppendTimestamp(b, time.Now())
	b = msgpack.AppendMap(b, 3)

	b = msgpack.AppendString(b, "a")
	b = msgpack.AppendBool(b, true)

	b = msgpack.AppendString(b, "b")
	b = msgpack.AppendInt(b, 123)

	b = msgpack.AppendString(b, "c")
	b = msgpack.AppendFloat64(b, 456.789)

	fmt.Println(len(b), ":", b)
	fmt.Println(msgpack.GetMsgpackValueLength(b))
	fmt.Println(msgpack.GetMsgpackValueLength(b[1:]))
	fmt.Println(msgpack.GetMsgpackValueLength(b[9:]))
	fmt.Println(msgpack.GetMsgpackValueLength(b[15:]))

	m := Value(b[15:])

	for k, v := range m.Map() {
		fmt.Println(k, "=", v)
	}

	// Output: TODO
}
