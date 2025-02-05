package fluentlog

import (
	"io"
	"testing"
)

func BenchmarkLogger(b *testing.B) {
	log := NewLogger(io.Discard)
	b.ResetTimer()

	for range b.N {
		_ = log.Info("hello world",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
			"foo", "bar",
		)
	}
}
