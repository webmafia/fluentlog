package main

import (
	"io"
	"runtime"
	"testing"

	"go.uber.org/zap"

	"github.com/webmafia/fluentlog"
)

func BenchmarkMessage(b *testing.B) {
	b.Run("Fluentlog", func(b *testing.B) {
		inst, err := fluentlog.NewInstance(io.Discard)

		if err != nil {
			b.Fatal(err)
		}

		log := inst.Logger()
		b.ResetTimer()

		for range b.N {
			log.Info("The quick brown fox jumps over the lazy dog")
		}
	})

	b.Run("Zap", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		b.ResetTimer()

		for range b.N {
			log.Info("The quick brown fox jumps over the lazy dog")
		}
	})

	b.Run("ZapSugar", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		sugar := log.Sugar()
		b.ResetTimer()

		for range b.N {
			sugar.Infow("The quick brown fox jumps over the lazy dog")
		}
	})
}

func Benchmark10Strings(b *testing.B) {
	b.Run("Fluentlog", func(b *testing.B) {
		inst, err := fluentlog.NewInstance(io.Discard)

		if err != nil {
			b.Fatal(err)
		}

		log := inst.Logger()
		b.ResetTimer()

		for range b.N {
			log.Info("The quick brown fox jumps over the lazy dog",
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
	})

	b.Run("Zap", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		b.ResetTimer()

		for range b.N {
			log.Info("The quick brown fox jumps over the lazy dog",
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
				zap.String("foo", "bar"),
			)
		}
	})

	b.Run("ZapSugar", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		sugar := log.Sugar()
		b.ResetTimer()

		for range b.N {
			sugar.Infow("The quick brown fox jumps over the lazy dog",
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
	})
}

func Benchmark10Ints(b *testing.B) {
	b.Run("Fluentlog", func(b *testing.B) {
		inst, err := fluentlog.NewInstance(io.Discard)

		if err != nil {
			b.Fatal(err)
		}

		log := inst.Logger()
		b.ResetTimer()

		for range b.N {
			log.Info("The quick brown fox jumps over the lazy dog",
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
			)
		}
	})

	b.Run("Zap", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		b.ResetTimer()

		for range b.N {
			log.Info("The quick brown fox jumps over the lazy dog",
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
				zap.Int("foo", 123456),
			)
		}
	})

	b.Run("ZapSugar", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		sugar := log.Sugar()
		b.ResetTimer()

		for range b.N {
			sugar.Infow("The quick brown fox jumps over the lazy dog",
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
				"foo", 123456,
			)
		}
	})
}

func BenchmarkParallell(b *testing.B) {
	b.Run("Fluentlog", func(b *testing.B) {
		inst, err := fluentlog.NewInstance(io.Discard, fluentlog.Options{
			BufferSize:    runtime.GOMAXPROCS(0) * 2,
			WriteBehavior: fluentlog.Loose,
		})

		if err != nil {
			b.Fatal(err)
		}

		log := inst.Logger()
		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				log.Info("The quick brown fox jumps over the lazy dog")
			}
		})
	})

	b.Run("Zap", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel)
		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				log.Info("The quick brown fox jumps over the lazy dog")
			}
		})
	})

	b.Run("ZapSugar", func(b *testing.B) {
		log := newZapLogger(zap.DebugLevel).Sugar()
		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				log.Info("The quick brown fox jumps over the lazy dog")
			}
		})
	})
}
