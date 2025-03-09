package forward

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"time"

	_ "unsafe"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/forward/transport"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

type ServerSession struct {
	serv     *Server
	conn     net.Conn
	iter     *msgpack.Iterator
	write    msgpack.Buffer
	user     []byte
	timeConn time.Time
	trans    transport.TransportPhase
	id       uint64
}

//go:linkname newServerSession forward.newServerSession
func newServerSession(s *Server, conn net.Conn) ServerSession {
	iter := s.iterPool.Get(conn)
	wBuf := s.bufPool.Get()

	return ServerSession{
		serv:     s,
		conn:     conn,
		iter:     iter,
		write:    msgpack.Buffer{Buffer: wBuf},
		timeConn: time.Now(),
	}
}

func (ss *ServerSession) authenticate(ctx context.Context) (err error) {
	var nonceAuth [48]byte

	if _, err = rand.Read(nonceAuth[:]); err != nil {
		return
	}

	nonce, auth := fast.BytesToString(nonceAuth[:24]), fast.BytesToString(nonceAuth[24:])

	if !ss.serv.opt.PasswordAuth {
		auth = ""
	}

	if err = ss.writeHelo(nonce, auth); err != nil {
		return
	}

	salt, cred, err := ss.readPing(ctx, nonce, auth)

	if err != nil {
		ss.writePong(nonce, salt, "", false, err.Error())
		return
	}

	if err = ss.writePong(salt, nonce, cred.SharedKey, true, ""); err != nil {
		return
	}

	ss.user = append(ss.user[:0], cred.Username...)

	return
}

func (ss *ServerSession) TotalRead() int {
	return ss.iter.TotalRead()
}

func (ss *ServerSession) Buffered() int {
	return ss.iter.Buffered()
}

func (ss *ServerSession) Username() string {
	return fast.BytesToString(ss.user)
}

func (ss *ServerSession) ID() uint64 {
	return ss.id
}

func (ss *ServerSession) Log(str string, args ...any) {
	if len(args) > 0 {
		str = fmt.Sprintf(str, args...)
	}

	log.Printf("%04d: %s", ss.id, str)
}

func (ss *ServerSession) initTransportPhase() {
	ss.trans.Init(&ss.serv.iterPool, &ss.serv.gzipPool, func(chunk string) (err error) {
		ss.write.WriteMapHeader(1)
		ss.write.WriteString("ack")
		ss.write.WriteString(chunk)

		_, err = ss.write.WriteTo(ss.conn)
		return
	})
}

func (ss *ServerSession) Next(e *transport.Entry) (err error) {
	ss.conn.SetReadDeadline(time.Now().Add(time.Second))

	return ss.trans.Next(ss.iter, e)
}

func (ss *ServerSession) Close() error {
	ss.serv.iterPool.Put(ss.iter)
	ss.serv.bufPool.Put(ss.write.Buffer)
	return ss.conn.Close()
}

func (ss *ServerSession) Rewind() {
	ss.iter.Rewind()
}
