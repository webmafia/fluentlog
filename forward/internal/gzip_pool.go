package internal

import (
	"bufio"
	"compress/flate"
	"io"
	"sync"
	"unsafe"

	"github.com/klauspost/compress/gzip"
)

type GzipPool struct {
	pool sync.Pool
}

type reader struct {
	_  gzip.Header
	_  flate.Reader
	br *bufio.Reader
}

func (g *GzipPool) Acquire(r io.Reader) (*gzip.Reader, error) {
	if inst, ok := g.pool.Get().(*gzip.Reader); ok {
		if err := inst.Reset(r); err != nil && err != io.EOF {
			return nil, err
		}

		return inst, nil
	}

	return gzip.NewReader(r)
}

func (g *GzipPool) Release(r *gzip.Reader) (err error) {
	g.reset(r)
	g.pool.Put(r)
	return
}

func (*GzipPool) reset(r *gzip.Reader) {
	p := (*reader)(unsafe.Pointer(r))

	if p.br != nil {
		p.br.Reset(nil)
	}
}
