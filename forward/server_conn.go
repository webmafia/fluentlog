package forward

import (
	"fmt"
	"log"
	"net"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type ServerConn struct {
	serv *Server
	conn net.Conn
	r    msgpack.Reader
	w    msgpack.Writer
}

func (s *ServerConn) Handle(fn func(*buffer.Buffer) error) (err error) {
	defer s.conn.Close()

	if err = s.handshakePhase(); err != nil {
		return
	}

	if err = s.transportPhase(); err != nil {
		return
	}

	// s.r.Release(0)

	// for {
	// 	// s.r.Release(0)

	// 	arr, err := s.r.Read()

	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Println(arr.Type(), ":", arr.String())
	// }

	// -----------------------------------------------------------------------------------

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

func (s *ServerConn) handshakePhase() (err error) {
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

	return
}

// Once the connection becomes transport phase, client can send events to servers, in one event mode of:
//   - Message Mode (single message)
//   - Forward Mode (an array of messages)
//   - PackedForward Mode (an array of messages sent as binary)
//   - CompressedPackedForward Mode (an array of messages sent as compressed binary)
func (s *ServerConn) transportPhase() (err error) {
	s.r.ResetReleasePoint()

	for {
		s.r.Release()

		var (
			arr msgpack.Value
			tag string
			val msgpack.Value
		)

		if arr, err = s.r.Read(); err != nil {
			return err
		}

		arrLen := arr.Len()

		// Abort early if invalid data
		if arr.Type() != types.Array || arrLen < 2 || arrLen > 4 {
			return fmt.Errorf("unexpected array length: %d", arrLen)
		}

		if tag, err = s.r.ReadStr(); err != nil {
			return
		}

		if val, err = s.r.ReadHead(); err != nil {
			return
		}

		switch val.Type() {

		case types.Ext, types.Int, types.Uint:
			if err = s.messageMode(tag, val, arrLen); err != nil {
				return
			}

		case types.Array:
			if err = s.forwardMode(tag, val); err != nil {
				return
			}

		case types.Bin:
			if err = s.packedForwardMode(tag, val); err != nil {
				return
			}

		}
	}

	return
}

// The Message Mode has the following format:
//
//	[
//	  "tag.name",               // 1. tag
//	  1441588984,               // 2. time
//	  {"message": "bar"},       // 3. record
//	  {"chunk": "<<UniqueId>>"} // 4. option (optional)
//	]
func (s *ServerConn) messageMode(tag string, ts msgpack.Value, arrLen int) (err error) {
	if arrLen < 3 {
		return ErrInvalidEntry
	}

	if ts, err = s.r.ReadFull(ts); err != nil {
		return
	}

	if ts.Timestamp().IsZero() {
		return ErrInvalidEntry
	}

	var rec msgpack.Value

	if rec, err = s.r.Read(); err != nil {
		return
	}

	if rec.Type() != types.Map {
		return ErrInvalidEntry
	}

	if rec, err = s.r.ReadFull(rec); err != nil {
		return
	}

	if arrLen == 4 {
		if _, err = s.r.Read(); err != nil {
			return
		}
	}

	return s.entry(tag, ts, rec)
}

// The Forward Mode has the following format:
//
//	[
//	  "tag.name",                         // 1. tag
//	  [                                   // 2. array of entries
//	    [1441588984, {"message": "foo"}],
//	    [1441588985, {"message": "bar"}],
//	    [1441588986, {"message": "baz"}]
//	  ],
//	  {"chunk": "<<UniqueId>>"}           // 3. options (optional)
//	]
func (s *ServerConn) forwardMode(tag string, arr msgpack.Value) (err error) {
	var (
		entry msgpack.Value
		ts    msgpack.Value
		rec   msgpack.Value
	)

	arrLen := arr.Len()

	for range arrLen {
		s.r.Release()

		if entry, err = s.r.Read(); err != nil {
			return
		}

		if entry.Type() != types.Array || entry.Len() != 2 {
			return ErrInvalidEntry
		}

		if ts, err = s.r.Read(); err != nil {
			return
		}

		if rec, err = s.r.Read(); err != nil {
			return
		}

		if rec.Type() != types.Map {
			return ErrInvalidEntry
		}

		if rec, err = s.r.ReadFull(rec); err != nil {
			return
		}

		if err = s.entry(tag, ts, rec); err != nil {
			return
		}
	}

	return
}

// The PackedForward Mode has the following format:
//
//	[
//	  "tag.name",                   // 1. tag
//	  "<<MessagePackEventStream>>", // 2. binary (bin) field of concatenated entries
//	  {"chunk": "<<UniqueId>>"}     // 3. options (optional)
//	]
func (s *ServerConn) packedForwardMode(tag string, v msgpack.Value) (err error) {
	gzip, err := s.isGzip()

	if err != nil {
		return
	}

	if gzip {
		return s.compressedPackedForwardMode(tag, v)
	}

	target := v.Len() + s.r.Total()

	for s.r.Total() < target {
		if v, err = s.r.Read(); err != nil {
			return
		}

		if v.Type() != types.Array {
			return ErrInvalidEntry
		}

		if err = s.forwardMode(tag, v); err != nil {
			return
		}
	}

	if s.r.Total() != target {
		err = ErrInvalidEntry
	}

	return
}

// The CompressedPackedForward Mode has the following format:
//
//	[
//	  "tag.name",                                     // 1. tag
//	  "<<CompressedMessagePackEventStream>>",         // 2. binary (bin) field of concatenated entries
//	  {"compressed": "gzip", "chunk": "<<UniqueId>>"} // 3. options with "compressed" (required)
//	]
func (s *ServerConn) compressedPackedForwardMode(tag string, bin msgpack.Value) (err error) {
	return fmt.Errorf("%w: compressed (gzip) stream", ErrNotSupported)
}

func (s *ServerConn) isGzip() (ok bool, err error) {
	magicNumbers, err := s.r.Peek(3)

	if err != nil {
		return
	}

	ok = (magicNumbers[0] == 0x1f &&
		magicNumbers[1] == 0x8b &&
		magicNumbers[2] == 8)

	return
}

func (s *ServerConn) entry(tag string, ts, rec msgpack.Value) (err error) {
	log.Println(tag, ts, rec)
	for k, v := range rec.Map() {
		log.Println("  ", k, "=", v)
	}
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
