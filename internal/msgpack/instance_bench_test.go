package msgpack

import (
	"sync/atomic"
	"testing"
)

func BenchmarkCloseChannel(b *testing.B) {
	ch := make(chan struct{})
	b.ResetTimer()

	for range b.N {
		select {
		case <-ch:
		default:
		}
	}
}

func BenchmarkCloseBoolean(b *testing.B) {
	var closed atomic.Bool
	b.ResetTimer()

	for range b.N {
		_ = closed.Swap(true)
	}
}
