package transport

import (
	"fmt"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

var _ Mode = (*ForwardMode)(nil)

type ForwardMode struct {
	t          *TransportPhase
	tag        []byte
	items      int
	hasOptions bool
}

// Rewind implements Mode.
func (m *ForwardMode) Rewind(iter *msgpack.Iterator) {
	iter.Rewind()
}

// String implements Mode.
func (m *ForwardMode) String() string {
	return "ForwardMode"
}

func (m *ForwardMode) Enter(iter *msgpack.Iterator, e *Entry, items int, hasOptions bool) error {
	iter.SetManualFlush(true)
	m.tag = append(m.tag[:0], e.Tag...)
	m.items = items
	m.hasOptions = hasOptions
	return m.t.changeMode(m, iter, e)
}

// Next implements Mode.
func (m *ForwardMode) Next(iter *msgpack.Iterator, e *Entry) (err error) {
	iter.Flush()

	if m.items <= 0 {
		return m.t.messageMode.Enter(iter, e)
	}

	// 0) Array of 2 items
	if err = iter.NextExpectedType(types.Array); err != nil {
		return m.t.error("array_head", err)
	}

	if items := iter.Items(); items != 2 {
		return m.t.error("array_head", fmt.Errorf("unexpected array length: expected %d, got %d", 2, items))
	}

	// 1) Timestamp
	if err = iter.NextExpectedType(types.Ext, types.Int, types.Uint); err != nil {
		return m.t.errorNoEof("time", err)
	}

	e.Tag = fast.BytesToString(m.tag)
	e.Timestamp = iter.Time()

	// 2) Record
	if err = iter.NextExpectedType(types.Map); err != nil {
		return m.t.errorNoEof("record", err)
	}

	e.Record = iter

	m.items--
	return
}

// Leave implements Mode.
func (m *ForwardMode) Leave(iter *msgpack.Iterator) (err error) {
	if m.hasOptions {
		err = m.t.handleOptions(iter)
	}

	return
}
