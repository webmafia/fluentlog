package gzip

import (
	"sync"

	"github.com/webmafia/fast/bufio"
)

type Pool struct {
	pool sync.Pool
}

func (pool *Pool) Get(br bufio.BufioReader) (r *Reader, err error) {
	var ok bool

	if r, ok = pool.pool.Get().(*Reader); ok {
		return r, r.Reset(br)
	}

	return NewReader(br)
}

func (pool *Pool) Put(r *Reader) {
	r.Reset(nil)
	pool.pool.Put(r)
}
