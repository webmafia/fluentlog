package fluentlog

import "testing"

func BenchmarkLogger(b *testing.B) {
	log := NewLogger(nil)
	b.ResetTimer()

	for range b.N {
		_ = log.Log("hello world",
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
