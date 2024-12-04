package forward

import (
	"context"
	"log"
	"net"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Server struct {
	opt ServerOptions
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

func (s *Server) Listen(ctx context.Context, addr string) (err error) {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", addr)

	if err != nil {
		return
	}

	defer listener.Close()

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
			if err := sc.Handle(); err != nil {
				log.Println(err)
			}
		}()
	}
}
