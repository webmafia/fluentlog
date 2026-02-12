package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/webmafia/fluentlog/forward"
	"github.com/webmafia/fluentlog/forward/transport"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
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
		Address: "localhost:24224",
		Auth: forward.StaticAuthServer(forward.Credentials{
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

		var i int

		for {
			if err = ss.Next(&e); err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					if i > 0 {
						fmt.Printf("Received %d messages\n", i)
						i = 0
					}
					continue
				}

				break
			}

			numFields := e.Record.Items()
			log.Println(e.Timestamp, e.Tag, "- received entry of", numFields, "fields")
			rec := e.Record

			for range numFields {
				if !rec.Next() {
					return rec.Error()
				}

				key := rec.Any()

				if !rec.Next() {
					return rec.Error()
				}

				fmt.Printf("   %s = ", key)

				if rec.Type() == types.Array {
					numFields := e.Record.Items()

					for i := range numFields {
						if !rec.Next() {
							return rec.Error()
						}

						fmt.Printf("\n      %d = %s", i, rec.Any())
					}

					fmt.Print("\n")
				} else {
					fmt.Println(rec.Any())
				}
			}

			i++

			// fmt.Printf("Received %d messages\r", i)
		}

		fmt.Print("\n")

		if i > 0 {
			fmt.Printf("Received %d messages\n", i)
		}

		return
	})
}
