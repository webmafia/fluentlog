package forward

import (
	"fmt"

	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

func (ss *ServerSession) toMessageMode(e *Entry, hasOptions bool) (err error) {
	if hasOptions {
		if err = ss.handleOptions(ss.iter); err != nil {
			return
		}
	}

	ss.mode = MessageMode
	ss.next = ss.modes.messageMode.next
	return ss.next(ss.iter, e)
}

type messageMode struct {
	ss   *ServerSession
	iter *msgpack.Iterator
}

func (m *messageMode) next(iter *msgpack.Iterator, e *Entry) (err error) {
	if err = iter.NextExpectedType(types.Array); err != nil {
		return m.ss.error("array_head", err)
	}

	evLen := iter.Items()

	// Abort early if invalid data
	if evLen < 2 || evLen > 4 {
		return m.ss.error("array_head", fmt.Errorf("unexpected array length: %d", evLen))
	}

	// 1) Tag
	if err = iter.NextExpectedType(types.Str); err != nil {
		return m.ss.errorNoEof("tag", err)
	}

	if iter.Len() < minTagLen {
		return m.ss.error("tag", fmt.Errorf("too short tag (%d chars), must be min %d chars", iter.Len(), minTagLen))
	}

	if iter.Len() > maxTagLen {
		return m.ss.error("tag", fmt.Errorf("too long tag (%d chars), must be max %d chars", iter.Len(), maxTagLen))
	}
	e.Tag = iter.Str()

	// 2) Time or Entries (Array / Bin / Str)
	if !iter.Next() {
		return m.ss.errorNoEof("time_or_entries", iter.Error())
	}

	switch iter.Type() {

	case types.Ext, types.Int, types.Uint:
		// MessageMode - keep going

	case types.Array:
		return m.ss.toForwardMode(iter, e, iter.Items(), evLen == 3)

	case types.Bin:
		limitR := iter.Reader()
		isGzip, err := isGzip(limitR)

		if err != nil {
			return m.ss.error("is_gzip", err)
		}

		iter.SetManualFlush(false)

		if isGzip {
			return m.ss.toCompressedPackedForwardMode(iter, e, limitR)
		}

		return m.ss.toPackedForwardMode(iter, e, limitR)

	default:
		return m.ss.error("time_or_entries", ErrInvalidEntry)

	}

	e.Timestamp = iter.Time()

	// 3) Record
	if err = iter.NextExpectedType(types.Map); err != nil {
		return m.ss.errorNoEof("record", err)
	}

	e.Record = iter

	// 4) Options
	if evLen == 4 {
		return m.ss.handleOptions(iter)
	}

	return
}
