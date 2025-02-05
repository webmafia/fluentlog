package fluentlog

import (
	"fmt"
	"runtime"
	"testing"
)

// func BenchmarkStackTrace(b *testing.B) {
// 	frames := runtime.Stack()
// }

func ExampleStackTrace() {
	var buf [1024]byte
	n := runtime.Stack(buf[:], false)

	fmt.Println(string(buf[:n]))

	// Output: TODO
}

func BenchmarkStackTrace(b *testing.B) {
	b.Run("New", func(b *testing.B) {
		for range b.N {
			_ = StackTrace()
		}
	})

	b.Run("New_and_AppendKeyValue", func(b *testing.B) {
		var buf []byte

		for range b.N {
			trace := StackTrace()
			buf, _ = trace.AppendKeyValue(buf[:0], "")
		}
	})
}
