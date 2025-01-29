package forward

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

type ServerConn struct {
	serv *Server
	conn net.Conn
	r    *msgpack.Iterator
	w    msgpack.Writer
}

func (s *ServerConn) String() string {
	_, port, _ := strings.Cut(s.conn.RemoteAddr().String(), ":")
	return port
}

func (s *ServerConn) log(str string, args ...any) {
	log.Println("client", s.String(), "|", fmt.Sprintf(str, args...))
}

func (s *ServerConn) Handle(fn func(*buffer.Buffer) error) (err error) {
	defer s.conn.Close()

	s.log("connected")

	if err = s.handshakePhase(); err != nil {
		return
	}

	if err = s.transportPhase(); err != nil {
		return
	}

	return
}

func (s *ServerConn) handshakePhase() (err error) {
	s.log("initializing handshake phase...")

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

	s.log("authenticated")

	return
}

// Once the connection becomes transport phase, client can send events to servers, in one event mode of:
//   - Message Mode (single message)
//   - Forward Mode (an array of messages)
//   - PackedForward Mode (an array of messages sent as binary)
//   - CompressedPackedForward Mode (an array of messages sent as compressed binary)
func (s *ServerConn) transportPhase() (err error) {
	s.log("initializing transport phase...")

	for {
		s.r.ResetReleasePoint()
		s.r.Release()

		// 0) Array of 2-4 items
		if err = s.r.NextExpectedType(types.Array); err != nil {
			return
		}
		evLen := s.r.Items()

		// Abort early if invalid data
		if evLen < 2 || evLen > 4 {
			return fmt.Errorf("unexpected array length: %d", evLen)
		}

		// 1) Tag
		if err = s.r.NextExpectedType(types.Str); err != nil {
			return
		}
		tag := s.r.Str()

		s.r.SetReleasePoint()

		// 2) Time or Entries (Array / Bin / Str)
		if !s.r.Next() {
			return io.ErrUnexpectedEOF
		}
		evLen -= 2

		switch s.r.Type() {

		case types.Ext, types.Int, types.Uint:
			if err = s.messageMode(tag, s.r.Time(), evLen); err != nil {
				return
			}

			evLen--

		case types.Array:
			if err = s.forwardMode(tag, s.r.Items()); err != nil {
				return
			}

		case types.Bin:
			if err = s.binary(tag); err != nil {
				return
			}

		default:
			return ErrInvalidEntry

		}

		// Optional options map
		if evLen == 1 {
			if err = s.ack(); err != nil {
				return
			}
		}
	}
}

// The Message Mode has the following format:
//
//	[
//	  "tag.name",               // 1. tag
//	  1441588984,               // 2. time
//	  {"message": "bar"},       // 3. record
//	  {"chunk": "<<UniqueId>>"} // 4. option (optional)
//	]
func (s *ServerConn) messageMode(tag string, ts time.Time, evLen int) (err error) {
	s.log("Message Mode")

	if evLen < 1 {
		return ErrInvalidEntry
	}

	if ts.IsZero() {
		return ErrInvalidEntry
	}

	// 3) Record
	if err = s.r.NextExpectedType(types.Map); err != nil {
		return
	}
	rec := s.r.Value()

	if err = s.r.Error(); err != nil {
		return
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
func (s *ServerConn) forwardMode(tag string, arrLen int) (err error) {
	s.log("Forward Mode")

	for range arrLen {
		s.r.Release()

		if err = s.iterateEntry(s.r, tag); err != nil {
			return
		}
	}

	return
}

func (s *ServerConn) binary(tag string) (err error) {
	isGzip, err := s.isGzip()

	if err != nil {
		return
	}

	if isGzip {
		return s.compressedPackedForwardMode(tag)
	}

	return s.packedForwardMode(tag)
}

// The PackedForward Mode has the following format:
//
//	[
//	  "tag.name",                   // 1. tag
//	  "<<MessagePackEventStream>>", // 2. binary (bin) field of concatenated entries
//	  {"chunk": "<<UniqueId>>"}     // 3. options (optional)
//	]
func (s *ServerConn) packedForwardMode(tag string) (err error) {
	s.log("PackedForward Mode")

	iter := s.serv.iterPool.Get()
	defer s.serv.iterPool.Put(iter)

	iter.Reset(s.r.BinReader())

	for {
		iter.Release()

		if err = s.iterateEntry(iter, tag); err != nil {
			return
		}
	}
}

// The CompressedPackedForward Mode has the following format:
//
//	[
//	  "tag.name",                                     // 1. tag
//	  "<<CompressedMessagePackEventStream>>",         // 2. binary (bin) field of concatenated entries
//	  {"compressed": "gzip", "chunk": "<<UniqueId>>"} // 3. options with "compressed" (required)
//	]
func (s *ServerConn) compressedPackedForwardMode(tag string) (err error) {
	s.log("CompressedPackedForward Mode")

	r, err := gzip.NewReader(s.r.BinReader())

	if err != nil {
		return
	}

	defer r.Close()

	iter := s.serv.iterPool.Get()
	defer s.serv.iterPool.Put(iter)

	iter.Reset(r)

	for {
		iter.Release()

		if err = s.iterateEntry(iter, tag); err != nil {
			return
		}
	}
}

// An Entry has the following format:
//
//	[
//	  1441588984,               // 1. time
//	  {"message": "bar"}        // 2. record
//	]
func (s *ServerConn) iterateEntry(iter *msgpack.Iterator, tag string) (err error) {

	// 0) Array of 2 items
	if err = iter.NextExpectedType(types.Array); err != nil {
		return
	}
	if items := iter.Items(); items != 2 {
		return ErrInvalidEntry
	}

	// 1) Timestamp
	if err = iter.NextExpectedType(types.Ext, types.Int, types.Uint); err != nil {
		return
	}
	ts := iter.Time()

	// 2) Record
	if err = iter.NextExpectedType(types.Map); err != nil {
		return
	}
	rec := iter.Value()

	if err = iter.Error(); err != nil {
		return
	}

	return s.entry(tag, ts, rec)
}

func (s *ServerConn) isGzip() (ok bool, err error) {
	if s.r.Len() < 3 {
		return
	}

	magicNumbers, err := s.r.Peek(3)

	if err != nil {
		return
	}

	ok = (magicNumbers[0] == 0x1f &&
		magicNumbers[1] == 0x8b &&
		magicNumbers[2] == 8)

	return
}

func (s *ServerConn) entry(tag string, ts time.Time, rec msgpack.Value) (err error) {
	log.Println(tag, ts, rec)
	for k, v := range rec.Map() {
		log.Println("  ", k, "=", v)
	}
	return
}

// Iterate options to find "chunk" value, and send ack back to client.
func (s *ServerConn) ack() (err error) {
	if err = s.r.NextExpectedType(types.Map); err != nil {
		return
	}

	mapLen := s.r.Items()
	s.r.ResetReleasePoint()

	for range mapLen {
		s.r.Release()

		if err = s.r.NextExpectedType(types.Str); err != nil {
			return
		}
		key := s.r.Str()

		if !s.r.Next() {
			if err = s.r.Error(); err != nil {
				return
			}

			return io.ErrUnexpectedEOF
		}

		if key != "chunk" {
			// log.Println("skipped", key, "=", val)
			continue
		}

		chunk := s.r.Str()
		s.w.WriteMapHeader(1)
		s.w.WriteString("ack")
		s.w.WriteString(chunk)

		s.log("ack %s", chunk)

		if err = s.w.Flush(); err != nil {
			return
		}
	}

	return
}
