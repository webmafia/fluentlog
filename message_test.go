package fluentlog

import (
	"fmt"
	"testing"
)

func Example() {
	msg := NewMessage()
	msg.AddField("foo", "bar")
	msg.AddField("baz", "test")

	for k, v := range msg.Fields() {
		fmt.Println(k, v)
	}

	// for i := 0; i < 32; i++ {
	// 	fmt.Println(msg.NumFields(), len(msg.buf))
	// 	msg.incNumFields()
	// }

	// Output: TODO
}

func BenchmarkXxx(b *testing.B) {
	msg := NewMessage()

	b.ResetTimer()

	for range b.N {
		msg.write(0x92)
		msg.Reset()
	}
}
