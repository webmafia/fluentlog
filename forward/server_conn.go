package forward

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"iter"
	"log"
	"net"
	"strings"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fast/bufio"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

type ServerConn struct {
	serv      *Server
	conn      net.Conn
	r         *msgpack.Iterator
	w         msgpack.Writer
	state     *buffer.Buffer
	user, tag string
}

func (s *ServerConn) Username() string {
	return s.user
}

func (s *ServerConn) Tag() string {
	return s.tag
}

func (s *ServerConn) TotalRead() int {
	return s.r.TotalRead()
}

func (s *ServerConn) String() string {
	_, port, _ := strings.Cut(s.conn.RemoteAddr().String(), ":")
	return port
}

func (s *ServerConn) log(str string, args ...any) {
	log.Println("client", s.String(), "|", fmt.Sprintf(str, args...))
}

func (s *ServerConn) handle(ctx context.Context, handler func(c *ServerConn) error) (err error) {
	defer s.conn.Close()

	s.log("connected")

	if err = s.handshakePhase(ctx); err != nil {
		return
	}

	return handler(fast.NoescapeVal(s))
}

func (s *ServerConn) handshakePhase(ctx context.Context) (err error) {
	s.log("initializing handshake phase...")

	s.r.LockBuffer()
	defer s.r.UnlockBuffer()

	var nonceAuth [48]byte

	if _, err = rand.Read(nonceAuth[:]); err != nil {
		return
	}

	nonce, auth := fast.BytesToString(nonceAuth[:24]), fast.BytesToString(nonceAuth[24:])

	if !s.serv.opt.PasswordAuth {
		auth = ""
	}

	if err = s.writeHelo(nonce, auth); err != nil {
		return
	}

	salt, cred, err := s.readPing(ctx, nonce, auth)

	if err != nil {
		s.writePong(nonce, salt, "", false, err.Error())
		return
	}

	if err = s.writePong(salt, nonce, cred.SharedKey, true, ""); err != nil {
		return
	}

	s.user = s.stateString(cred.Username)
	s.log("authenticated")

	return
}

func (s *ServerConn) Entries() iter.Seq2[time.Time, *msgpack.Iterator] {
	return func(yield func(time.Time, *msgpack.Iterator) bool) {
		if err := s.transportPhase(yield); err != nil {
			log.Println(err)
		}
	}
}

func (s *ServerConn) stateString(str string) string {
	start := len(s.state.B)
	s.state.B = append(s.state.B, str...)
	end := len(s.state.B)

	return fast.BytesToString(s.state.B[start:end])
}

// Once the connection becomes transport phase, client can send events to servers, in one event mode of:
//   - Message Mode (single message)
//   - Forward Mode (an array of messages)
//   - PackedForward Mode (an array of messages sent as binary)
//   - CompressedPackedForward Mode (an array of messages sent as compressed binary)
func (s *ServerConn) transportPhase(yield func(time.Time, *msgpack.Iterator) bool) (err error) {
	s.log("initializing transport phase...")

	more := true

	for more {
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

		if s.r.Len() < 1 {
			return fmt.Errorf("too short tag (%d chars), must be min %d chars", s.r.Len(), 1)
		}

		if s.r.Len() > 64 {
			return fmt.Errorf("too long tag (%d chars), must be max %d chars", s.r.Len(), 64)
		}
		s.tag = s.stateString(s.r.Str())

		// 2) Time or Entries (Array / Bin / Str)
		if !s.r.Next() {
			return io.ErrUnexpectedEOF
		}
		evLen -= 2

		switch s.r.Type() {

		case types.Ext, types.Int, types.Uint:
			more, err = s.messageMode(yield, s.r.Time(), evLen)
			evLen--

		case types.Array:
			more, err = s.forwardMode(yield, s.r.Items())

		case types.Bin:
			more, err = s.binary(yield)

		default:
			return ErrInvalidEntry

		}

		if err != nil && err != io.EOF {
			return
		}

		// Optional options map
		if evLen == 1 {
			if err = s.ack(); err != nil {
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
func (s *ServerConn) messageMode(yield func(time.Time, *msgpack.Iterator) bool, ts time.Time, evLen int) (more bool, err error) {
	s.log("Message Mode")

	if evLen < 1 {
		return false, ErrInvalidEntry
	}

	if ts.IsZero() {
		return false, ErrInvalidEntry
	}

	// 3) Record
	if err = s.r.NextExpectedType(types.Map); err != nil {
		return
	}

	return yield(ts, s.r), nil
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
func (s *ServerConn) forwardMode(yield func(time.Time, *msgpack.Iterator) bool, arrLen int) (more bool, err error) {
	s.log("Forward Mode")

	for range arrLen {
		if more, err = s.iterateEntry(yield, s.r); !more {
			return
		}
	}

	return
}

func (s *ServerConn) binary(yield func(time.Time, *msgpack.Iterator) bool) (more bool, err error) {
	r := s.r.Reader()
	isGzip, err := s.isGzip(r)

	if err != nil {
		return
	}

	if isGzip {
		return s.compressedPackedForwardMode(yield, r)
	}

	return s.packedForwardMode(yield, r)
}

// The PackedForward Mode has the following format:
//
//	[
//	  "tag.name",                   // 1. tag
//	  "<<MessagePackEventStream>>", // 2. binary (bin) field of concatenated entries
//	  {"chunk": "<<UniqueId>>"}     // 3. options (optional)
//	]
func (s *ServerConn) packedForwardMode(yield func(time.Time, *msgpack.Iterator) bool, br *bufio.LimitedReader) (more bool, err error) {
	s.log("PackedForward Mode")

	iter := s.serv.iterPool.Get(br)
	defer s.serv.iterPool.Put(iter)

	for {
		if more, err = s.iterateEntry(yield, iter); !more {
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
func (s *ServerConn) compressedPackedForwardMode(yield func(time.Time, *msgpack.Iterator) bool, br *bufio.LimitedReader) (more bool, err error) {
	s.log("CompressedPackedForward Mode")

	// r, err := gzip.NewReader(br)
	// r, err := gzip.NewReader(br)
	r, err := s.serv.gzipPool.Get(br)

	if err != nil {
		return
	}

	defer s.serv.gzipPool.Put(r)

	iter := s.serv.iterPool.Get(r)
	defer s.serv.iterPool.Put(iter)

	for {
		if more, err = s.iterateEntry(yield, iter); !more {
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
func (s *ServerConn) iterateEntry(yield func(time.Time, *msgpack.Iterator) bool, iter *msgpack.Iterator) (more bool, err error) {

	// 0) Array of 2 items
	if err = iter.NextExpectedType(types.Array); err != nil {
		return
	}
	if items := iter.Items(); items != 2 {
		return false, ErrInvalidEntry
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

	return yield(ts, iter), nil
}

func (*ServerConn) isGzip(r *bufio.LimitedReader) (ok bool, err error) {
	magicNumbers, err := r.Peek(3)

	if err != nil {
		return
	}

	ok = (magicNumbers[0] == 0x1f &&
		magicNumbers[1] == 0x8b &&
		magicNumbers[2] == 8)

	return
}

// Iterate options to find "chunk" value, and send ack back to client.
func (s *ServerConn) ack() (err error) {
	if err = s.r.NextExpectedType(types.Map); err != nil {
		return
	}

	mapLen := s.r.Items()

	for range mapLen {
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
			s.r.Skip()
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
