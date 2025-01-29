package forward

import (
	"context"
	"crypto/tls"
	"log"
	"net"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Server struct {
	opt      ServerOptions
	bufPool  buffer.Pool
	iterPool msgpack.IterPool
}

type ServerOptions struct {
	Address   string
	Hostname  string
	Tls       *tls.Config // E.g. from golang.org/x/crypto/acme/autocert
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
		iterPool: msgpack.IterPool{
			BufMaxSize: 16 * 1024, // 16 kB
		},
	}
}

func (s *Server) Listen(ctx context.Context, fn func(*buffer.Buffer) error) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", s.opt.Address)

	if err != nil {
		return
	}

	if s.opt.Tls != nil {
		listener = tls.NewListener(listener, s.opt.Tls)
	}

	heartbeat, err := s.listenHeartbeat(ctx)

	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		heartbeat.Close()
		listener.Close()
		log.Println("Closed server")
	}()

	log.Println("Listening on", s.opt.Address)

	for {
		conn, err := listener.Accept()

		if err != nil {
			return err
		}

		go func() {
			iter := s.iterPool.Get()
			defer s.iterPool.Put(iter)

			iter.Reset(conn)

			wBuf := s.bufPool.Get()
			defer s.bufPool.Put(wBuf)

			sc := ServerConn{
				serv: s,
				conn: conn,
				r:    iter,
				w:    msgpack.NewWriter(conn, wBuf),
			}

			if err := sc.Handle(fn); err != nil {
				log.Println(err)
			}
		}()
	}
}
