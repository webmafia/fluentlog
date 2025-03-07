package forward

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"

	"github.com/webmafia/fast"
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
	HandleError  func(err error)
	Auth         AuthServer
	PasswordAuth bool
	ReadTimeout  time.Duration
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

	if opt.HandleError == nil {
		opt.HandleError = func(err error) {}
	}

	return &Server{
		opt: opt,
		// iterPool: msgpack.IterPool{
		// 	BufMaxSize: 16 * 1024, // 16 kB
		// },
	}
}

func (s *Server) Listen(ctx context.Context, handler func(ctx context.Context, ss *ServerSession) error) (err error) {
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
			ss := newServerSession(s, conn)
			defer ss.Close()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			if err := ss.authenticate(ctx); err != nil {
				s.opt.HandleError(err)
				return
			}

			ss.initTransportPhase()

			if err := handler(ctx, fast.NoescapeVal(&ss)); err != nil && err != io.EOF {
				s.opt.HandleError(err)
			}
		}()
	}
}
