package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/webmafia/fluentlog/forward"
	"github.com/webmafia/fluentlog/forward/transport"
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
		Address:      "localhost:24284",
		PasswordAuth: true,
		Auth: forward.StaticAuthServer(forward.Credentials{
			Username:  "foo",
			Password:  "bar",
			SharedKey: "secret",
		}),
		HandleError: func(err error) {
			log.Println("client error:", err)
		},
		ReadTimeout: 1 * time.Second,
	})

	return serv.Listen(ctx, func(ctx context.Context, ss *forward.ServerSession) (err error) {
		var e transport.Entry

		log.Println("connected")
		defer log.Println("disconnected")

		for {
			if err = ss.Next(&e); err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					log.Println("timed out:", err)
					continue
				}

				return
			}

			numFields := e.Record.Items()
			log.Println(ss.Username(), e.Tag, e.Timestamp)
			log.Println("received entry of", numFields, "fields")
			rec := e.Record

			for range numFields {
				if !rec.Next() {
					return rec.Error()
				}

				key := rec.Any()

				if !rec.Next() {
					return rec.Error()
				}

				val := rec.Any()

				log.Println("  ", key, "=", val)

				if key == "message" && val == "hello 199" {
					log.Println("tada")
				}
			}
		}

		return
	})
}
