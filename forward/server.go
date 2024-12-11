package forward

import (
	"context"
	"log"
	"net"

	"github.com/valyala/bytebufferpool"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Server struct {
	opt     ServerOptions
	bufPool bytebufferpool.Pool
}

type ServerOptions struct {
	Hostname  string
	SharedKey func(clientHostname string) (sharedKey []byte, err error)
}

func SharedKey(sharedKey []byte) func(clientHostname string) (sharedKey []byte, err error) {
	return func(_ string) ([]byte, error) {
		return sharedKey, nil
	}
}

func NewServer(opt ServerOptions) *Server {
	if opt.SharedKey == nil {
		opt.SharedKey = func(clientHostname string) (sharedKey []byte, err error) { return nil, nil }
	}

	return &Server{
		opt: opt,
	}
}

func (s *Server) Listen(ctx context.Context, addr string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", addr)

	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		listener.Close()
		log.Println("Closed listener")
	}()

	log.Println("Listening on", addr)

	for {
		conn, err := listener.Accept()

		if err != nil {
			return err
		}

		sc := ServerConn{
			serv: s,
			conn: conn,
			r:    msgpack.NewReader(conn, make([]byte, 4096)),
			w:    msgpack.NewWriter(conn, make([]byte, 4096)),
		}

		go func() {
			if err := sc.Handle(fn); err != nil {
				log.Println(err)
			}
		}()
	}
}
