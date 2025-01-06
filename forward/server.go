package forward

import (
	"context"
	"log"
	"net"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Server struct {
	opt     ServerOptions
	bufPool buffer.Pool
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

func (s *Server) Listen(ctx context.Context, addr string, fn func(*buffer.Buffer) error) (err error) {
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

		go func() {
			rBuf := s.bufPool.Get()
			defer s.bufPool.Put(rBuf)

			wBuf := s.bufPool.Get()
			defer s.bufPool.Put(wBuf)

			sc := ServerConn{
				serv: s,
				conn: conn,
				r:    msgpack.NewReader(conn, rBuf, 16*1024),
				w:    msgpack.NewWriter(conn, wBuf),
			}

			if err := sc.Handle(fn); err != nil {
				log.Println(err)
			}
		}()
	}
}
