package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/valyala/bytebufferpool"
	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/forward"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := startServer(ctx); err != nil {
		log.Println(err)
	}
}

func startServer(ctx context.Context) (err error) {
	serv := forward.NewServer(forward.ServerOptions{
		SharedKey: forward.SharedKey([]byte("secret")),
	})

	// addr := "localhost:24224"
	addr := "localhost:24284"

	return serv.Listen(ctx, addr, func(b *bytebufferpool.ByteBuffer) error {
		msg := fluentlog.MsgFromBuf(b.B)
		log.Println(msg.Tag(), msg.Time())

		for k, v := range msg.Fields().Map() {
			log.Println(k.String(), v.String())
		}

		return nil
	})

	return
}
