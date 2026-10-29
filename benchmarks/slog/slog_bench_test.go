package slogbench

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func BenchmarkSlogOperations(b *testing.B) {
	attrs := []any{"foo", "bar", "count", 42}
	typedAttrs := []slog.Attr{
		slog.String("foo", "bar"),
		slog.Int("count", 42),
	}
	largeTypedAttrs := []slog.Attr{
		slog.String("a", "1"),
		slog.String("b", "2"),
		slog.String("c", "3"),
		slog.String("d", "4"),
		slog.String("e", "5"),
		slog.String("f", "6"),
		slog.String("g", "7"),
		slog.String("h", "8"),
		slog.String("i", "9"),
		slog.String("j", "10"),
	}
	anyValue := struct {
		Name  string
		Count int
	}{
		Name:  "bar",
		Count: 42,
	}
	ctx := context.Background()

	handlers := map[string]slog.Handler{
		"TextHandler": slog.NewTextHandler(io.Discard, nil),
		"JSONHandler": slog.NewJSONHandler(io.Discard, nil),
	}

	for name, handler := range handlers {
		b.Run(name, func(b *testing.B) {
			base := slog.New(handler)

			b.Run("Info/message_only", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.Info("hello")
				}
			})

			b.Run("Log/message_only", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.Log(ctx, slog.LevelInfo, "hello")
				}
			})

			b.Run("Info/key_values", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.Info("hello", attrs...)
				}
			})

			b.Run("Log/key_values", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.Log(ctx, slog.LevelInfo, "hello", attrs...)
				}
			})

			b.Run("LogAttrs/typed", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.LogAttrs(ctx, slog.LevelInfo, "hello", typedAttrs...)
				}
			})

			b.Run("LogAttrs/typed_10", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.LogAttrs(ctx, slog.LevelInfo, "hello", largeTypedAttrs...)
				}
			})

			b.Run("LogAttrs/any", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.LogAttrs(ctx, slog.LevelInfo, "hello",
						slog.Any("payload", anyValue),
					)
				}
			})

			b.Run("With/reused_logger", func(b *testing.B) {
				b.ReportAllocs()
				log := base.With(attrs...)
				for b.Loop() {
					log.Info("hello")
				}
			})

			b.Run("With/per_call", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					base.With(attrs...).Info("hello")
				}
			})

			b.Run("WithAttrs/reused_handler", func(b *testing.B) {
				b.ReportAllocs()
				log := slog.New(handler.WithAttrs(typedAttrs))
				for b.Loop() {
					log.Info("hello")
				}
			})
		})
	}
}
