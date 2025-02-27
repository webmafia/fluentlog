package main

import (
	"fmt"
	"time"

	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
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

func validateMessage(iter *msgpack.Iterator) (err error) {
	if err = iter.NextExpectedType(types.Array); err != nil {
		return
	}

	if iter.Items() != 3 {
		return fmt.Errorf("expected array length %d, got %d", 3, iter.Items())
	}

	if err = iter.NextExpectedType(types.Str); err != nil {
		return
	}

	if str := iter.Str(); str != "foobar" {
		return fmt.Errorf("expected string '%s', got '%s'", "foobar", str)
	}

	if err = iter.NextExpectedType(types.Ext); err != nil {
		return
	}

	iter.Skip()

	if err = iter.NextExpectedType(types.Map); err != nil {
		return
	}

	if iter.Items() != 1 {
		return fmt.Errorf("expected map length %d, got %d", 1, iter.Items())
	}

	if err = iter.NextExpectedType(types.Str); err != nil {
		return
	}

	if str := iter.Str(); str != "message" {
		return fmt.Errorf("expected string '%s', got '%s'", "message", str)
	}

	if err = iter.NextExpectedType(types.Str); err != nil {
		return
	}

	if str := iter.Str(); str != "deadbeaf" {
		return fmt.Errorf("expected string '%s', got '%s'", "deadbeaf", str)
	}

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
