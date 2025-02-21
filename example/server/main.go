package main

import (
	"context"
	"log"
	"os"
	"os/signal"

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
		Address:      "localhost:24284",
		PasswordAuth: true,
		Auth: forward.StaticAuthServer(forward.Credentials{
			Username:  "foo",
			Password:  "bar",
			SharedKey: "secret",
		}),
	})

	return serv.Listen(ctx, func(ctx context.Context, c *forward.ServerConn) error {
		for ts, rec := range c.Entries() {
			numFields := rec.Items()
			log.Println(c.Username(), c.Tag(), ts)
			log.Println("received entry of", numFields, "fields")

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
			}
		}

		return nil
	})
}
