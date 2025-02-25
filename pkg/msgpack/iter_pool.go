package msgpack

import (
	"io"
	"sync"

	"github.com/webmafia/fast/ringbuf"
)

// memory waste.
type IterPool struct {
	pool sync.Pool
}

func (p *IterPool) Get(r io.Reader) (iter *Iterator) {
	var ok bool

	if iter, ok = p.pool.Get().(*Iterator); ok {
		iter.Reset(r)
		return
	}

	return &Iterator{r: ringbuf.NewReader(r)}
}

func (p *IterPool) Put(iter *Iterator) {
	iter.Reset(nil)
	p.pool.Put(iter)
}
