package forward

import (
	"github.com/webmafia/fast/ringbuf"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

func (ss *ServerSession) toPackedForwardMode(iter *msgpack.Iterator, e *Entry, r *ringbuf.LimitedReader) error {
	return nil
}
