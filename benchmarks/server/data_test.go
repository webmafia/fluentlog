package main

import (
	"time"

	"github.com/webmafia/fluentlog/pkg/msgpack"
)

// For our benchmark, we need a valid msgpack payload.
// In this example we use a Message Mode event:
//
//	[ "foobar", <timestamp>, {"message": "test"} ]
func createMessage() (b []byte) {
	b = msgpack.AppendArrayHeader(b, 3)
	b = msgpack.AppendString(b, "foobar")
	b = msgpack.AppendTimestamp(b, time.Now())
	b = msgpack.AppendMapHeader(b, 1)
	b = msgpack.AppendString(b, "message")
	b = msgpack.AppendString(b, "deadbeaf")

	return
}

func createPayload(size int) (b []byte, numMessages int) {
	msg := createMessage()
	numMessages = size / len(msg)
	b = make([]byte, 0, numMessages*len(msg))

	for range numMessages {
		b = append(b, msg...)
	}

	return
}
