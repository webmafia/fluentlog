package fluentlog

import (
	"io"
	"log"
	"sync"
	"testing"
)

func BenchmarkLogger(b *testing.B) {
	inst, err := NewInstance(io.Discard, Options{
		BufferSize: 8,
	})

	if err != nil {
		b.Fatal(err)
	}

	log := inst.Logger()
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
	inst, err := NewInstance(io.Discard)

	if err != nil {
		b.Fatal(err)
	}

	log := inst.Logger()
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
	inst, err := NewInstance(io.Discard)

	if err != nil {
		b.Fatal(err)
	}

	log := inst.Logger()
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

func ExampleLogger_Recover() {
	inst, err := NewInstance(io.Discard)

	if err != nil {
		log.Fatal(err)
	}

	log := inst.Logger()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Recover()

		panic("aaaaaahh")
	}()

	wg.Wait()
}
