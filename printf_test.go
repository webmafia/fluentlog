package fluentlog

import (
	"fmt"
	"testing"
)

func Benchmark_countFmtArgs(b *testing.B) {
	b.Run("easy", func(b *testing.B) {
		for b.Loop() {
			_ = countFmtArgs("a=%d b=%d c=%d")
		}
	})

	b.Run("medium", func(b *testing.B) {
		for b.Loop() {
			_ = countFmtArgs("a=%d b=%[3]d c=%d")
		}
	})

	b.Run("hard", func(b *testing.B) {
		for b.Loop() {
			_ = countFmtArgs("start %% %[3]d %#+08x %-10.5s %[4]d %06d %d %d end")
		}
	})
}

func Example_countFmtArgs() {
	fmt.Println(countFmtArgs("start %% %[3]d %#+08x %-10.5s %[4]d %06d %d %d end"))
	// Output: 7
}
