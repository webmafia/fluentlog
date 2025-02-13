package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/webmafia/fluentlog/forward"
	"github.com/webmafia/fluentlog/internal/msgpack"
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
	})

	return serv.Listen(ctx, func(tag string, ts time.Time, iter *msgpack.Iterator, numFields int) error {
		log.Println(tag, ts)
		log.Println("received entry of", numFields, "fields")

		for range numFields {
			if !iter.Next() {
				return iter.Error()
			}

			key := iter.Any()

			if !iter.Next() {
				return iter.Error()
			}

			val := iter.Any()

			log.Println("  ", key, "=", val)
		}

		return nil
	})
}
