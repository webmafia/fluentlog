package identifier

import (
	"math/rand"
	"sync/atomic"
	"time"
)

// Non-thread-safe ID generator.
type Generator struct {
	seq uint32
}

// Create an ID generator. The generator is NOT thread-safe.
func NewGenerator() Generator {
	return Generator{
		seq: rand.Uint32(),
	}
}

func (g *Generator) ID() (id ID) {
	id = newID(time.Now(), g.seq)
	g.seq++
	return
}

func (g *Generator) IDFromTime(ts time.Time) (id ID) {
	id = newID(ts, g.seq)
	g.seq++
	return
}

// Thread-safe ID generator.
type AtomicGenerator struct {
	seq uint32
}

// Create an atomic ID generator. The generator is thread-safe.
func NewAtomicGenerator() AtomicGenerator {
	return AtomicGenerator{
		seq: rand.Uint32(),
	}
}

func (g *AtomicGenerator) ID() ID {
	return newID(time.Now(), atomic.AddUint32(&g.seq, 1))
}

func (g *AtomicGenerator) IDFromTime(ts time.Time) ID {
	return newID(ts, atomic.AddUint32(&g.seq, 1))
}
