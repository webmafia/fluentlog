package msgpack

import (
	"io"
	"sync"
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

	it := NewIterator(r)

	return &it
}

func (p *IterPool) Put(iter *Iterator) {
	iter.Reset(nil)
	p.pool.Put(iter)
}
