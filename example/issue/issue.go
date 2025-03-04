package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := simulateIssue(ctx); err != nil {
		log.Println(err)
	}
}

func simulateIssue(ctx context.Context) (err error) {
	const addr = "127.0.0.1:1234"
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", addr)

	if err != nil {
		return
	}

	defer listener.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, err := listener.Accept()

		if err != nil {
			return
		}

		defer conn.Close()

		var buf [1]byte

		for {
			conn.SetReadDeadline(time.Now().Add(time.Second))

			log.Println("awaiting data...")
			_, err := conn.Read(buf[:])

			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					log.Println("one second has passed")
					continue
				}

				log.Println("server: client error", err)
				return
			}

			log.Println("data received:", buf)
		}
	}()

	var dial net.Dialer

	conn, err := dial.DialContext(ctx, "tcp", addr)

	if err != nil {
		return
	}

	defer conn.Close()

	if _, err = conn.Write([]byte{1}); err != nil {
		return
	}

	time.Sleep(time.Second)

	if _, err = conn.Write([]byte{1}); err != nil {
		return
	}

	time.Sleep(time.Second)
	conn.Close()

	<-ctx.Done()
	listener.Close()
	wg.Wait()

	return
}
