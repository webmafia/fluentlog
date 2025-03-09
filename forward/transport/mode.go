package transport

import (
	"fmt"

	"github.com/webmafia/fluentlog/pkg/msgpack"
)

type Mode interface {
	fmt.Stringer
	Next(iter *msgpack.Iterator, e *Entry) error
	Leave(iter *msgpack.Iterator) error
	Rewind(iter *msgpack.Iterator)
}
