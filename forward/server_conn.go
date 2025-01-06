package forward

import (
	"log"
	"net"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type ServerConn struct {
	serv *Server
	conn net.Conn
	r    msgpack.Reader
	w    msgpack.Writer
}

func (s *ServerConn) Handle(fn func(*buffer.Buffer) error) (err error) {
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

	// for {
	// 	s.r.Release(0)

	// 	arr, err := s.r.Read()

	// 	if err != nil {
	// 		return err
	// 	}

	// 	if arr.Type() != types.Array || arr.Len() < 2 || arr.Len() > 4 {
	// 		return fmt.Errorf("unexpected array length: %d", arr.Len())
	// 	}

	// 	log.Println("array length received:", arr.Len())

	// 	// Item 1: Tag
	// 	tag, err := s.r.Read()

	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Println("tag received:", tag)

	// 	// Should be either a binary (event stream), or a timestamp (single event)
	// 	typ, err := s.r.PeekType()

	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Println("type:", typ)

	// 	if typ == msgpack.TypeArray {
	// 		return s.forward(tag, fn)
	// 	}

	// 	if typ == msgpack.TypeBinary {
	// 		return s.packedForward(tag, fn)
	// 	}

	// 	// Item 2: Time
	// 	if _, err = s.r.ReadTimestamp(); err != nil {
	// 		return err
	// 	}

	// 	// Item 3: Record
	// 	if err = s.r.SkipMap(); err != nil {
	// 		return err
	// 	}

	// 	buf := s.serv.bufPool.Get()

	// 	if _, err = buf.Write(s.r.ConsumedBuffer()); err != nil {
	// 		return err
	// 	}

	// 	if err = fn(buf); err != nil {
	// 		return err
	// 	}

	// 	// Item 4: Option
	// 	if arrLen == 4 {
	// 		if err = s.r.SkipMap(); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	return
}

// func (s *ServerConn) forward(tag string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
// 	pos := s.r.Pos()

// 	arrLen, err := s.r.ReadArrayHeader()

// 	if err != nil {
// 		return
// 	}

// 	log.Println("entries:", arrLen)

// 	for range arrLen {
// 		s.r.ReleaseAfter(pos)

// 		if err = s.forwardEntry(tag, fn); err != nil {
// 			return
// 		}
// 	}

// 	return
// }

// func (s *ServerConn) forwardEntry(tag string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
// 	arrLen, err := s.r.ReadArrayHeader()

// 	if err != nil {
// 		return err
// 	}

// 	if arrLen != 2 {
// 		return errors.New("invalid entry")
// 	}

// 	log.Println("still going strong")

// 	buf := s.serv.bufPool.Get()
// 	buf.B = msgpack.AppendArray(buf.B, 3)
// 	buf.B = msgpack.AppendString(buf.B, tag)

// 	log.Println(buf.B)

// 	v1, err := s.r.ReadRaw()

// 	if err != nil {
// 		return err
// 	}

// 	log.Println(v1)
// 	log.Println(s.r.PeekType())

// 	v2, err := s.r.ReadRaw()

// 	if err != nil {
// 		return err
// 	}

// 	log.Println(v2)

// 	buf.B = append(buf.B, v1...)
// 	buf.B = append(buf.B, v2...)

// 	log.Println(buf.B)

// 	return fn(buf)
// }

// func (s *ServerConn) packedForward(tag string, fn func(*bytebufferpool.ByteBuffer) error) (err error) {
// 	binLen, err := s.r.SkipBinaryHeader()

// 	if err != nil {
// 		return
// 	}

// 	pos := s.r.Pos()

// 	gzipHead, err := s.r.PeekBytes(2)

// 	if err != nil {
// 		return
// 	}

// 	if gzipHead[0] == 0x1f && gzipHead[1] == 0x8b {
// 		return errors.New("compressed stream is not yet supported")
// 	}

// 	var read int

// 	for read < binLen {
// 		if err = s.r.ReleaseTo(pos); err != nil {
// 			return err
// 		}

// 		if err = s.forwardEntry(tag, fn); err != nil {
// 			return
// 		}

// 		read += s.r.Pos() - pos
// 	}

// 	return
// }
