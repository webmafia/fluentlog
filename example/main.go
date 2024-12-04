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

	if err := startServer(ctx); err != nil {
		log.Println(err)
	}

	if err := startClient(ctx); err != nil {
		log.Println(err)
	}
}

func startClient(ctx context.Context) (err error) {
	cli := forward.NewClient("localhost:24224", forward.ClientOptions{
		SharedKey: []byte("secret"),
	})

	if err = cli.Connect(ctx); err != nil {
		return
	}

	msg := fluentlog.NewMessage("foo.bar", time.Now())
	msg.AddField("foo", 123)
	msg.AddField("bar", "baz")

	if err = cli.Send(msg); err != nil {
		return
	}

	log.Println("sent message")
	time.Sleep(time.Second)

	return
}

func startServer(ctx context.Context) (err error) {
	serv := forward.NewServer(forward.ServerOptions{
		SharedKey: forward.SharedKey([]byte("secret")),
	})

	go serv.Listen(ctx, "localhost:24224")

	return
}
