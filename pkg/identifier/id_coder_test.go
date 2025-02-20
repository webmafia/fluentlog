package identifier

import (
	"testing"
)

func BenchmarkID_AppendText(b *testing.B) {
	var buf []byte
	g := NewGenerator()
	id := g.ID()
	b.ResetTimer()

	for range b.N {
		buf, _ = id.AppendText(buf[:0])
	}
}

func TestMultiplierInverse(t *testing.T) {
	var multiplier = multiplier
	var invMultiplier = invMultiplier

	// Multiplication on uint64 is performed modulo 2^64.
	product := multiplier * invMultiplier

	if product != 1 {
		t.Errorf("Expected multiplier * invMultiplier mod 2^64 to equal 1, got %x", product)
	}
}
