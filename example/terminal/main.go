package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/forward"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := startClient(ctx); err != nil {
		log.Println(err)
	}
}

func startClient(ctx context.Context) (err error) {
	inst, err := fluentlog.NewInstance(forward.NewAsciiFormatter(os.Stdout), fluentlog.Options{
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
	l.Info("hello")
	l.Info("world")
	return
}
