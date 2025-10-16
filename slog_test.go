package fluentlog

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func ExampleLogger_SlogHandler() {
	inst, err := NewInstance(io.Discard)

	if err != nil {
		panic(err)
	}

	l := inst.Logger()
	log := slog.New(l.SlogHandler())

	log.Info("foobar")
}

func BenchmarkSlog(b *testing.B) {
	inst, err := NewInstance(io.Discard)

	if err != nil {
		b.Fatal(err)
	}

	b.Run("Fluentlog", func(b *testing.B) {
		log := inst.Logger()

		for b.Loop() {
			log.Info("foobar", "foo", "bar")
		}
	})

	b.Run("FluentlogViaSlog", func(b *testing.B) {
		log := inst.Logger()
		l := slog.New(log.SlogHandler())

		for b.Loop() {
			l.Info("foobar", "foo", "bar")
		}
	})

	b.Run("SlogDiscard", func(b *testing.B) {
		l := slog.New(DiscardHandler)

		for b.Loop() {
			l.Info("foobar", "foo", "bar")
		}
	})
}

var DiscardHandler slog.Handler = discardHandler{}

type discardHandler struct{}

func (dh discardHandler) Enabled(ctx context.Context, level slog.Level) bool   { return true }
func (dh discardHandler) Handle(ctx context.Context, record slog.Record) error { return nil }
func (dh discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler             { return dh }
func (dh discardHandler) WithGroup(name string) slog.Handler                   { return dh }
