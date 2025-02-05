package fluentlog

import (
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

func Benchmark_appendKeyValue(b *testing.B) {
	var (
		buf []byte
		n   uint8
	)

	for range b.N {
		buf, n = appendKeyValue(buf[:0], "foo", "bar")
	}

	_ = n
}

func Benchmark_AppendAny(b *testing.B) {
	var buf []byte

	for range b.N {
		buf = msgpack.AppendString(buf[:0], "foo")
		buf = msgpack.AppendAny(buf[:0], "bar")
	}
}

func Benchmark_AppendString(b *testing.B) {
	var buf []byte

	for range b.N {
		buf = msgpack.AppendString(buf[:0], "foo")
		buf = msgpack.AppendString(buf[:0], "bar")
	}
}
