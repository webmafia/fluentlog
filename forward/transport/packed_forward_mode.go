package transport

import (
	"fmt"
	"io"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

var _ Mode = (*PackedForwardMode)(nil)

type PackedForwardMode struct {
	t          *TransportPhase
	iter       *msgpack.Iterator
	tag        []byte
	hasOptions bool
}

// Rewind implements Mode.
func (m *PackedForwardMode) Rewind(_ *msgpack.Iterator) {
	m.iter.Rewind()
}

// String implements Mode.
func (m *PackedForwardMode) String() string {
	return "PackedForwardMode"
}

func (m *PackedForwardMode) Enter(origIter *msgpack.Iterator, e *Entry, r io.Reader, hasOptions bool) (err error) {
	origIter.SetManualFlush(false)

	m.iter = m.t.iterPool.Get(r)
	m.tag = append(m.tag[:0], e.Tag...)
	m.hasOptions = hasOptions
	return m.t.changeMode(m, origIter, e)
}

// Next implements Mode.
func (m *PackedForwardMode) Next(origIter *msgpack.Iterator, e *Entry) (err error) {
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
func (m *PackedForwardMode) Leave(origIter *msgpack.Iterator) (err error) {
	m.t.iterPool.Put(m.iter)
	m.iter = nil

	if m.hasOptions {
		err = m.t.handleOptions(origIter)
	}

	return
}
