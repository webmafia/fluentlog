package transport

import (
	"fmt"
	"io"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal/gzip"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

type TransportPhase struct {
	iterPool    *msgpack.IterPool
	gzipPool    *gzip.Pool
	ack         func(chunk string) error
	mode        Mode
	messageMode MessageMode
	forwardMode ForwardMode
	packedMode  PackedForwardMode
	compMode    CompressedPackedForwardMode
}

func (t *TransportPhase) Init(iterPool *msgpack.IterPool, gzipPool *gzip.Pool, ack func(chunk string) error) {
	t.iterPool = iterPool
	t.gzipPool = gzipPool
	t.ack = ack
	t.messageMode.t = t
	t.forwardMode.t = t
	t.packedMode.t = t
	t.compMode.t = t
	t.mode = &t.messageMode
}

func (t *TransportPhase) Next(iter *msgpack.Iterator, e *Entry) error {
	return t.mode.Next(iter, fast.NoescapeVal(e))
}

func (t *TransportPhase) changeMode(mode Mode, iter *msgpack.Iterator, e *Entry) (err error) {
	if mode == t.mode {
		return fmt.Errorf("already in %s", mode)
	}

	if err = t.mode.Leave(iter); err != nil {
		return
	}

	t.mode = mode

	return t.Next(iter, e)
}

func (t *TransportPhase) handleOptions(iter *msgpack.Iterator) (err error) {
	if err = iter.NextExpectedType(types.Map); err != nil {
		return t.errorNoEof("ack", err)
	}

	for range iter.Items() {
		if err = iter.NextExpectedType(types.Str); err != nil {
			return t.errorNoEof("ack", err)
		}
		key := iter.Str()

		if !iter.Next() {
			if err = iter.Error(); err != nil {
				return t.errorNoEof("ack", err)
			}

			return t.errorNoEof("ack", io.ErrUnexpectedEOF)
		}

		if key != "chunk" {
			iter.Skip()
			continue
		}

		if err = t.ack(iter.Str()); err != nil {
			return t.error("ack", err)
		}
	}

	return
}

func (t *TransportPhase) error(op string, err any) error {
	if v, ok := err.(error); ok {
		if v == io.EOF {
			return v
		}

		return fmt.Errorf("%s, %s: %w", t.mode, op, v)
	}

	return fmt.Errorf("%s, %s: %v", t.mode, op, err)
}

func (t *TransportPhase) errorNoEof(op string, err any) error {
	if v, ok := err.(error); ok {
		if v == io.EOF {
			return io.ErrUnexpectedEOF
		}

		return fmt.Errorf("%s, %s: %w", t.mode, op, v)
	}

	return fmt.Errorf("%s, %s: %v", t.mode, op, err)
}

func (t *TransportPhase) Rewind(iter *msgpack.Iterator) {
	t.mode.Rewind(iter)
}
