package bench

import "testing"

func BenchmarkCopy(b *testing.B) {
	var (
		src = make([]byte, 4096)
		dst = make([]byte, 4096)
	)

	b.ResetTimer()

	for range b.N {
		_ = copy(dst, src)
	}
}

func BenchmarkAppend(b *testing.B) {
	var (
		src = make([]byte, 4096)
		dst = make([]byte, 4096)
	)

	b.ResetTimer()

	for range b.N {
		dst = dst[:0]
		dst = append(dst, src...)
	}
}
