package transport

import (
	"fmt"
	"io"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/ringbuf"
	"github.com/webmafia/fluentlog/internal/gzip"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

var _ Mode = (*CompressedPackedForwardMode)(nil)

type CompressedPackedForwardMode struct {
	t          *TransportPhase
	iter       *msgpack.Iterator
	gzip       *gzip.Reader
	tag        []byte
	hasOptions bool
}

// String implements Mode.
func (m *CompressedPackedForwardMode) String() string {
	return "CompressedPackedForwardMode"
}

func (m *CompressedPackedForwardMode) Enter(origIter *msgpack.Iterator, e *Entry, br ringbuf.RingBufferReader, hasOptions bool) (err error) {
	origIter.SetManualFlush(false)

	if m.gzip, err = m.t.gzipPool.Get(br); err != nil {
		return
	}

	m.iter = m.t.iterPool.Get(m.gzip)
	m.tag = append(m.tag[:0], e.Tag...)
	m.hasOptions = hasOptions

	return m.t.changeMode(m, origIter, e)
}

// Next implements Mode.
func (m *CompressedPackedForwardMode) Next(origIter *msgpack.Iterator, e *Entry) (err error) {
	m.iter.Flush()

	if err = m.iter.NextExpectedType(types.Array); err != nil {
		if err == io.EOF {
			return m.t.messageMode.Enter(origIter, e)
		}

		return m.t.error("array_head", err)
	}

	// 0) Array of 2 items
	if items := m.iter.Items(); items != 2 {
		return m.t.error("array_head", fmt.Errorf("unexpected array length: expected %d, got %d", 2, items))
	}

	// 1) Timestamp
	if err = m.iter.NextExpectedType(types.Ext, types.Int, types.Uint); err != nil {
		return m.t.errorNoEof("time", err)
	}

	e.Tag = fast.BytesToString(m.tag)
	e.Timestamp = m.iter.Time()

	// 2) Record
	if err = m.iter.NextExpectedType(types.Map); err != nil {
		return m.t.errorNoEof("record", err)
	}

	e.Record = m.iter

	return
}

// Leave implements Mode.
func (m *CompressedPackedForwardMode) Leave(origIter *msgpack.Iterator) (err error) {
	m.t.iterPool.Put(m.iter)
	m.t.gzipPool.Put(m.gzip)
	m.iter = nil
	m.gzip = nil

	if m.hasOptions {
		err = m.t.handleOptions(origIter)
	}

	return
}
