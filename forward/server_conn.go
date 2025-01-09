package forward

import (
	"fmt"
	"log"
	"net"
	"strings"

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
	for {
		s.r.ResetReleasePoint()
		s.r.Release()

		var (
			arr msgpack.Value
			tag string
			val msgpack.Value
		)

		if arr, err = s.r.Read(); err != nil {
			return err
		}

		evLen := arr.Len()

		// Abort early if invalid data
		if arr.Type() != types.Array || evLen < 2 || evLen > 4 {
			return fmt.Errorf("unexpected array length: %d", evLen)
		}

		if tag, err = s.r.ReadStr(); err != nil {
			return
		}

		s.r.SetReleasePoint()

		if val, err = s.r.ReadHead(); err != nil {
			return
		}

		evLen -= 2

		switch val.Type() {

		case types.Ext, types.Int, types.Uint:
			if err = s.messageMode(tag, val, evLen); err != nil {
				return
			}

			evLen--

		case types.Array:
			if err = s.forwardMode(tag, val); err != nil {
				return
			}

		case types.Bin:
			if err = s.packedForwardMode(tag, val); err != nil {
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
func (s *ServerConn) messageMode(tag string, ts msgpack.Value, evLen int) (err error) {
	if evLen < 1 {
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

func (s *ServerConn) ack() (err error) {
	var m msgpack.Value

	if m, err = s.r.Read(); err != nil {
		return
	}

	if m.Type() != types.Map {
		return ErrInvalidEntry
	}

	mapLen := m.Len()
	s.r.ResetReleasePoint()

	var (
		key string
		val msgpack.Value
	)

	for range mapLen {
		s.r.Release()

		if key, err = s.r.ReadStr(); err != nil {
			return
		}

		if val, err = s.r.Read(); err != nil {
			return
		}

		if typ := val.Type(); typ == types.Array || typ == types.Map {
			return ErrInvalidEntry
		}

		if key != "chunk" {
			// log.Println("skipped", key, "=", val)
			continue
		}

		chunk := val.Str()
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
