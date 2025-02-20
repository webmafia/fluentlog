package forward

import (
	"context"
	"crypto/tls"
	"log"
	"net"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/gzip"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

type Server struct {
	opt      ServerOptions
	bufPool  buffer.Pool
	iterPool msgpack.IterPool
	gzipPool gzip.Pool
}

type ServerOptions struct {
	Address      string
	Hostname     string
	Tls          *tls.Config // E.g. from golang.org/x/crypto/acme/autocert
	Auth         AuthServer
	PasswordAuth bool
}

func SharedKey(sharedKey []byte) func(clientHostname string) (sharedKey []byte, err error) {
	return func(_ string) ([]byte, error) {
		return sharedKey, nil
	}
}

func NewServer(opt ServerOptions) *Server {
	if opt.Auth == nil {
		opt.Auth = func(ctx context.Context, username string) (cred Credentials, err error) { return }
	}

	return &Server{
		opt: opt,
		iterPool: msgpack.IterPool{
			BufMaxSize: 16 * 1024, // 16 kB
		},
	}
}

func (s *Server) Listen(ctx context.Context, handler func(c *ServerConn) error) (err error) {
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
			iter := s.iterPool.Get(conn)
			defer s.iterPool.Put(iter)

			wBuf := s.bufPool.Get()
			defer s.bufPool.Put(wBuf)

			state := s.bufPool.Get()
			defer s.bufPool.Put(state)

			sc := ServerConn{
				serv:  s,
				conn:  conn,
				r:     iter,
				w:     msgpack.NewWriter(conn, wBuf),
				state: state,
			}

			if err := sc.handle(ctx, handler); err != nil {
				log.Println(err)
			}

			sc.log("received %d entries (totally %d bytes)", sc.entries, sc.r.TotalRead())
			sc.log("disconnected")
		}()
	}
}
