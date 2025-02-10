package fluentlog

import (
	"io"
	"testing"
)

func BenchmarkInstance_queueMessage(b *testing.B) {
	inst, err := NewInstance(io.Discard)

	if err != nil {
		b.Fatal(err)
	}

	log := inst.Logger()
	b.ResetTimer()

	for range b.N {
		buf := inst.bufPool.Get()
		log.inst.bufPool.Put(buf)
	}
}
