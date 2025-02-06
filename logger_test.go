package fluentlog

import (
	"io"
	"testing"
)

func BenchmarkLogger(b *testing.B) {
	log := NewLogger(NewInstance(io.Discard))
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

func BenchmarkSubLogger(b *testing.B) {
	log := NewLogger(NewInstance(io.Discard))
	l := log.With(
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
	b.ResetTimer()

	for range b.N {
		_ = l.Info("hello world")
	}
}

func BenchmarkLogger_With(b *testing.B) {
	log := NewLogger(NewInstance(io.Discard))
	b.ResetTimer()

	for range b.N {
		l := log.With(
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

		l.Release()
	}
}
