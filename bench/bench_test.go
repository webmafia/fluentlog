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
