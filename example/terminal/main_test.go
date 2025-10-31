package main

import (
	"io"
	"testing"

	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/forward"
)

func Benchmark(b *testing.B) {
	inst, err := fluentlog.NewInstance(forward.NewAsciiFormatter(io.Discard), fluentlog.Options{
		Tag:                 "foo.baz",
		WriteBehavior:       fluentlog.Block,
		BufferSize:          4,
		StackTraceThreshold: fluentlog.NOTICE,
	})

	if err != nil {
		return
	}

	defer inst.Close()

	l := fluentlog.NewLogger(inst)

	for b.Loop() {
		l.Info("hello")
	}
}
