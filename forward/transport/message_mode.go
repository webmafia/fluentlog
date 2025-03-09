package transport

import (
	"fmt"

	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

const (
	minTagLen = 1
	maxTagLen = 64
)

var _ Mode = (*MessageMode)(nil)

type MessageMode struct {
	t *TransportPhase
}

// Rewind implements Mode.
func (m *MessageMode) Rewind(iter *msgpack.Iterator) {
	iter.Rewind()
}

// String implements Mode.
func (m *MessageMode) String() string {
	return "MessageMode"
}

func (m *MessageMode) Enter(iter *msgpack.Iterator, e *Entry) error {
	iter.SetManualFlush(true)
	return m.t.changeMode(m, iter, e)
}

// Next implements Mode.
func (m *MessageMode) Next(iter *msgpack.Iterator, e *Entry) (err error) {
	iter.Flush()

	if err = iter.NextExpectedType(types.Array); err != nil {
		return m.t.error("array_head", err)
	}

	evLen := iter.Items()

	// Abort early if invalid data
	if evLen < 2 || evLen > 4 {
		return m.t.error("array_head", fmt.Errorf("unexpected array length: %d", evLen))
	}

	// 1) Tag
	if err = iter.NextExpectedType(types.Str); err != nil {
		return m.t.errorNoEof("tag", err)
	}

	if iter.Len() < minTagLen {
		return m.t.error("tag", fmt.Errorf("too short tag (%d chars), must be min %d chars", iter.Len(), minTagLen))
	}

	if iter.Len() > maxTagLen {
		return m.t.error("tag", fmt.Errorf("too long tag (%d chars), must be max %d chars", iter.Len(), maxTagLen))
	}
	e.Tag = iter.Str()

	// 2) Time or Entries (Array / Bin / Str)
	if !iter.Next() {
		return m.t.errorNoEof("time_or_entries", iter.Error())
	}

	switch iter.Type() {

	case types.Ext, types.Int, types.Uint:
		// MessageMode - keep going

	case types.Array:
		return m.t.forwardMode.Enter(iter, e, iter.Items(), evLen == 3)

	case types.Bin:
		limitR := iter.Reader()
		isGzip, err := isGzip(limitR)

		if err != nil {
			return m.t.error("is_gzip", err)
		}

		// iter.SetManualFlush(false)

		if isGzip {
			return m.t.compMode.Enter(iter, e, limitR, evLen == 3)
		}

		return m.t.packedMode.Enter(iter, e, limitR, evLen == 3)

	default:
		return m.t.error("time_or_entries", "invalid entry")

	}

	e.Timestamp = iter.Time()

	// 3) Record
	if err = iter.NextExpectedType(types.Map); err != nil {
		return m.t.errorNoEof("record", err)
	}

	e.Record = iter

	// 4) Options
	if evLen == 4 {
		return m.t.handleOptions(iter)
	}

	return
}

// Leave implements Mode.
func (m *MessageMode) Leave(iter *msgpack.Iterator) error {
	return nil
}
