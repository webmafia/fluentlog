package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/fallback"
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
	// addr := "localhost:24224"
	addr := "localhost:24284"

	cli := forward.NewClient(addr, forward.ClientOptions{
		Auth: forward.StaticAuthClient(forward.Credentials{
			Username:  "foo",
			Password:  "bar",
			SharedKey: "secret",
		}),
	})

	// _ = cli

	// f := filebuf.NewFileBuffer("log-buffer.bin")
	// defer f.Close()

	inst, err := fluentlog.NewInstance(cli, fluentlog.Options{
		Tag:                 "foo.baz",
		WriteBehavior:       fluentlog.Block,
		Fallback:            fallback.NewDirBuffer("fluentlog"),
		BufferSize:          4,
		StackTraceThreshold: fluentlog.NOTICE,
	})

	if err != nil {
		return
	}

	defer inst.Close()

	l := fluentlog.NewLogger(inst)

	sub := l.With(
		"valueFrom", "subLogger",
	)

	defer sub.Release()

	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	defer sub.Recover()

	// 	panic("aaaaaahh")
	// }()

	// wg.Wait()

	// cli.Connect(ctx)

	// sub.Error("woah, something happaned")

	for i := range 1_000_000 {
		// sub.Metrics("batch", i)
		sub.Infof("batch a: hello %d", i+1)
	}

	// time.Sleep(2 * time.Second)

	// for i := range 10 {
	// 	sub.Infof("batch a: hello %d", i+1)
	// }

	// time.Sleep(10 * time.Second)

	// if err = cli.Connect(ctx); err != nil {
	// 	return
	// }

	// time.Sleep(3 * time.Second)

	// msg := fluentlog.NewMessage("foo.bar", time.Now())
	// msg.AddField("foo", 123)
	// msg.AddField("bar", "baz")

	// if err = cli.Send(msg); err != nil {
	// 	return
	// }

	// log.Println("sent message")
	// time.Sleep(time.Second)

	return
}
