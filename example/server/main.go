package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/webmafia/fast/buffer"
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
		// Address: "localhost:24224",
		Address:   "localhost:24284",
		SharedKey: forward.SharedKey([]byte("secret")),
	})

	return serv.Listen(ctx, func(b *buffer.Buffer) error {
		log.Println(b.String())
		log.Println(b.Bytes())

		// msg := fluentlog.MsgFromBuf(b.B)
		// log.Println(msg.Tag(), msg.Time())

		// for k, v := range msg.Fields().Map() {
		// 	log.Println(k.String(), v.String())
		// }

		return nil
	})

	return
}
