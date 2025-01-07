package fluentlog

import (
	"fmt"
	"testing"
)

func BenchmarkTryWriteChannel(b *testing.B) {
	ch := make(chan int)

	tries := [...]int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}

	for _, i := range tries {
		b.Run(fmt.Sprintf("N%d", i), func(b *testing.B) {
			for range b.N {
				_ = tryWrite(ch, 1, i)
			}

			b.ReportMetric(float64(b.Elapsed())/float64(i*b.N), "ns/try")

		})
	}
}

func BenchmarkTryWriteChannel_Parallell(b *testing.B) {
	ch := make(chan int)

	tries := [...]int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}

	for _, i := range tries {
		b.Run(fmt.Sprintf("N%d", i), func(b *testing.B) {
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					_ = tryWrite(ch, 1, i)
				}
			})
		})
	}
}
