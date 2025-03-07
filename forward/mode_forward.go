package forward

import (
	"fmt"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

func (ss *ServerSession) toForwardMode(iter *msgpack.Iterator, e *Entry, items int, hasOptions bool) error {
	ss.mode = ForwardMode
	ss.next = ss.modes.forwardMode.next

	ss.modes.forwardMode.tag = append(ss.modes.forwardMode.tag[:0], e.Tag...)
	ss.modes.forwardMode.items = items
	ss.modes.forwardMode.hasOptions = hasOptions

	return ss.next(iter, e)
}

type forwardMode struct {
	ss         *ServerSession
	tag        []byte
	items      int
	hasOptions bool
}

func (m *forwardMode) next(iter *msgpack.Iterator, e *Entry) (err error) {
	if m.items <= 0 {
		return m.ss.toMessageMode(e, m.hasOptions)
	}

	// 0) Array of 2 items
	if err = iter.NextExpectedType(types.Array); err != nil {
		return m.ss.error("array_head", err)
	}

	if items := iter.Items(); items != 2 {
		return m.ss.error("array_head", fmt.Errorf("unexpected array length: expected %d, got %d", 2, items))
	}

	// 1) Timestamp
	if err = iter.NextExpectedType(types.Ext, types.Int, types.Uint); err != nil {
		return m.ss.errorNoEof("time", err)
	}

	e.Tag = fast.BytesToString(m.tag)
	e.Timestamp = iter.Time()

	// 2) Record
	if err = iter.NextExpectedType(types.Map); err != nil {
		return m.ss.errorNoEof("record", err)
	}

	e.Record = iter

	m.items--
	return
}
