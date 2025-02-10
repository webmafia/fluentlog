package fluentlog

import (
	"testing"
)

func Benchmark_appendStackTrace(b *testing.B) {
	var buf []byte

	for range b.N {
		buf, _ = appendStackTrace(buf[:0], 2)
	}

}
