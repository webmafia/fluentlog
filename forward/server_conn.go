package forward

import (
	"fmt"
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
		s.r.Release()

		arrLen, err := s.r.ReadArrayHeader()

		if err != nil {
			return err
		}

		if arrLen < 2 || arrLen > 4 {
			return fmt.Errorf("unexpected array length: %d", arrLen)
		}

		// Item 1: Tag
		tag, err := s.r.ReadString()

		if err != nil {
			return err
		}

		// Should be either a binary (event stream), or a timestamp (single event)
		typ, err := s.r.PeekType()

		if err != nil {
			return err
		}

		if typ == msgpack.TypeBinary {
			return s.handleStream(arrLen, tag)
		}

		// Item 2: Time
		if err = s.r.SkipTimestamp(); err != nil {
			return err
		}

		// Item 3: Record
		if err = s.r.SkipMap(); err != nil {
			return err
		}

		// Item 4: Option
		if arrLen == 4 {
			if err = s.r.SkipMap(); err != nil {
				return err
			}
		}
	}

	return
}

func (s *ServerConn) handleStream(arrLen int, tag string) (err error) {
	_ = s.r.Pos()
	return
}
