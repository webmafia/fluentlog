package forward

import (
	"log"
	"net"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

type ServerConn struct {
	serv *Server
	conn net.Conn
	r    msgpack.Reader
	w    msgpack.Writer
}

func (s *ServerConn) Handle() (err error) {
	defer s.conn.Close()

	nonce, err := s.writeHelo()

	if err != nil {
		return
	}

	salt, sharedKey, err := s.readPing(nonce[:])

	if err != nil {
		s.writePong(nonce[:], salt, sharedKey, false, err.Error())
		return
	}

	if err = s.writePong(nonce[:], salt, sharedKey, true, ""); err != nil {
		return
	}

	log.Println("server: client connected!")

	for {
		arrLen, err := s.r.ReadArrayHeader()

		if err != nil {
			return err
		}

	}

	return
}
