package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

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
	cli := forward.NewClient("localhost:24224", forward.ClientOptions{
		SharedKey: []byte("secret"),
	})

	l := fluentlog.NewLogger(cli)
	defer l.Close()

	// cli.Connect(ctx)

	for i := range 100 {
		l.Log("hello world",
			"count", i+1,
		)
	}

	// if err = cli.Connect(ctx); err != nil {
	// 	return
	// }

	time.Sleep(3 * time.Second)

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
