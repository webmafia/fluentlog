package transport

import (
	"time"

	"github.com/webmafia/fluentlog/pkg/msgpack"
)

type Entry struct {
	Tag       string
	Timestamp time.Time
	Record    *msgpack.Iterator
}
