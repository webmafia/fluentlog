package forward

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/valyala/bytebufferpool"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type ServerConn struct {
	serv *Server
	conn net.Conn
	r    msgpack.Reader
	w    msgpack.Writer
}

func (s *ServerConn) Handle(fn func(*bytebufferpool.ByteBuffer) error) (err error) {
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

		log.Println("array length received:", arrLen)

		// Item 1: Tag
		tag, err := s.r.ReadString()

		if err != nil {
			return err
		}

		log.Println("tag received:", tag)

		// Should be either a binary (event stream), or a timestamp (single event)
		typ, err := s.r.PeekType()

		if err != nil {
			return err
		}

		log.Println("type:", typ)

		if typ == msgpack.TypeArray {
			return s.forward(tag, fn)
		}

		if typ == msgpack.TypeBinary {
			return s.packedForward(tag, fn)
		}

		// Item 2: Time
		if _, err = s.r.ReadTimestamp(); err != nil {
			return err
		}

		// Item 3: Record
		if err = s.r.SkipMap(); err != nil {
			return err
		}

		buf := s.serv.bufPool.Get()

		if _, err = buf.Write(s.r.ConsumedBuffer()); err != nil {
			return err
		}

		if err = fn(buf); err != nil {
			return err
		}

		// Item 4: Option
		if arrLen == 4 {
			if err = s.r.SkipMap(); err != nil {
				return err
			}
		}
	}
}

func (s *ServerConn) forward(tag string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
	pos := s.r.Pos()

	arrLen, err := s.r.ReadArrayHeader()

	if err != nil {
		return
	}

	log.Println("entries:", arrLen)

	for range arrLen {
		if err = s.r.ReleaseTo(pos); err != nil {
			return
		}

		if err = s.forwardEntry(tag, fn); err != nil {
			return
		}
	}

	return
}

func (s *ServerConn) forwardEntry(tag string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
	arrLen, err := s.r.ReadArrayHeader()

	if err != nil {
		return err
	}

	if arrLen != 2 {
		return errors.New("invalid entry")
	}

	log.Println("still going strong")

	buf := s.serv.bufPool.Get()
	buf.B = msgpack.AppendArray(buf.B, 3)
	buf.B = msgpack.AppendString(buf.B, tag)

	log.Println(buf.B)

	v1, err := s.r.ReadRaw()

	if err != nil {
		return err
	}

	log.Println(v1)
	log.Println(s.r.PeekType())

	v2, err := s.r.ReadRaw()

	if err != nil {
		return err
	}

	log.Println(v2)

	buf.B = append(buf.B, v1...)
	buf.B = append(buf.B, v2...)

	log.Println(buf.B)

	return fn(buf)
}

func (s *ServerConn) packedForward(tag string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
	binLen, err := s.r.SkipBinaryHeader()

	if err != nil {
		return
	}

	pos := s.r.Pos()

	// r := io.LimitedReader{
	// 	R: &s.r,
	// 	N: int64(binLen),
	// }

	// s.r.ChangeReader(&r)

	// if err = s.r.ReleaseTo(pos); err != nil {
	// 	return
	// }

	gzipHead, err := s.r.PeekBytes(2)

	if err != nil {
		return
	}

	if gzipHead[0] == 0x1f && gzipHead[1] == 0x8b {
		return errors.New("compressed stream is not yet supported")
		// return s.compressedPackedForward(tag)
	}

	var read int

	for read < binLen {
		if err = s.r.ReleaseTo(pos); err != nil {
			return err
		}

		if err = s.forwardEntry(tag, fn); err != nil {
			return
		}

		read += s.r.Pos() - pos
	}

	return
}

// func (s *ServerConn) compressedPackedForward(tag string) (err error) {

// 	pos := s.r.Pos()

// 	if err = s.r.ReleaseTo(pos); err != nil {
// 		return
// 	}

// 	arrLen, err := s.r.ReadArrayHeader()

// 	if err != nil {

// 	}

// 	r, err := gzip.NewReader(&s.r)

// 	return errors.New("stream not implemented yet")
// }
