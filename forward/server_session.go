package forward

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	_ "unsafe"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/ringbuf"
	"github.com/webmafia/fluentlog/internal/gzip"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

const (
	minTagLen = 1
	maxTagLen = 64
)

type Entry struct {
	Tag       string
	Timestamp time.Time
	Record    *msgpack.Iterator
}

type EventMode uint8

// The Fluent Forwawrd protocol has four different event modes.
const (
	MessageMode EventMode = iota
	ForwardMode
	PackedForwardMode
	CompressedPackedForwardMode
)

type ServerSession struct {
	serv      *Server
	conn      net.Conn
	iter      *msgpack.Iterator
	origIter  *msgpack.Iterator
	write     msgpack.Buffer
	user, tag []byte
	timeConn  time.Time
	curItems  int32
	mode      EventMode
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

func (ss *ServerSession) Username() string {
	return fast.BytesToString(ss.user)
}

func (ss *ServerSession) Next(e *Entry) (err error) {
	ss.prepareForMessage()

	if !ss.iter.Next() {
		if err := ss.iter.Error(); err != io.EOF || ss.mode <= ForwardMode {
			return ss.error(err)
		}

		ss.resumeMessageMode()

		if !ss.iter.Next() {
			return ss.error(ss.iter.Error())
		}
	}

	// Options from previous call
	if ss.iter.Type() == types.Map {
		if err = ss.writeAck(); err != nil {
			return ss.error(err)
		}

		ss.resumeMessageMode()

		if !ss.iter.Next() {
			return ss.error(ss.iter.Error())
		}
	}

	if ss.mode == ForwardMode {
		if ss.curItems <= 0 {
			log.Println("ss.curItems reached zero")
			ss.resumeMessageMode()
		} else {
			goto forwardMode
		}
	}

	if ss.mode != MessageMode {
		goto packedForwardMode
	}

	// ss.prepareForMessage()

messageMode:
	// 0) Array of 2-4 items
	if ss.iter.Type() != types.Array {
		return ss.error(ErrInvalidEntry)
	}

	// Abort early if invalid data
	if evLen := ss.iter.Items(); evLen < 2 || evLen > 4 {
		return ss.error(fmt.Errorf("unexpected array length: %d", evLen))
	}

	// 1) Tag
	if err = ss.iter.NextExpectedType(types.Str); err != nil {
		return ss.error(err)
	}

	if ss.iter.Len() < minTagLen {
		return ss.error(fmt.Errorf("too short tag (%d chars), must be min %d chars", ss.iter.Len(), minTagLen))
	}

	if ss.iter.Len() > maxTagLen {
		return ss.error(fmt.Errorf("too long tag (%d chars), must be max %d chars", ss.iter.Len(), maxTagLen))
	}
	ss.tag = append(ss.tag[:0], ss.iter.Bin()...)

	// 2) Time or Entries (Array / Bin / Str)
	if !ss.iter.Next() {
		return ss.error(ss.iter.Error())
	}

	switch ss.iter.Type() {

	case types.Ext, types.Int, types.Uint:
		ss.mode = MessageMode
		goto entryRecord

	case types.Array:
		ss.mode = ForwardMode
		ss.curItems = int32(ss.iter.Items())

	case types.Bin:
		limitR := ss.iter.Reader()
		isGzip, err := isGzip(limitR)

		if err != nil {
			return ss.error(err)
		}

		ss.iter.SetManualFlush(false)

		if isGzip {
			gzip, err := ss.serv.gzipPool.Get(limitR)

			if err != nil {
				return ss.error(err)
			}

			ss.mode = CompressedPackedForwardMode
			ss.origIter, ss.iter = ss.iter, ss.serv.iterPool.Get(gzip)
		} else {
			ss.mode = PackedForwardMode
			ss.origIter, ss.iter = ss.iter, ss.serv.iterPool.Get(limitR)
		}

	default:
		return ss.error(ErrInvalidEntry)

	}

	if !ss.iter.Next() {
		return ss.error(ss.iter.Error())
	}

forwardMode:
	ss.curItems--

packedForwardMode:

	// ss.prepareForMessage()

	// 0) Array of 2 items
	if ss.iter.Type() != types.Array {
		f, _ := os.Create("debug.txt")
		ss.iter.DebugDump(f)
		f.Close()

		log.Println("ss.curItems", ss.curItems)
		return ss.error(fmt.Errorf("%w: expected %s, got %s", ErrInvalidEntry, types.Array, ss.iter.Type()))
	}

	if items := ss.iter.Items(); items != 2 {
		if items > 2 {
			ss.mode = MessageMode
			goto messageMode
		}

		return ss.error(fmt.Errorf("unexpected array length: expected %d, got %d", 2, items))
	}

	// 1) Timestamp
	if err = ss.iter.NextExpectedType(types.Ext, types.Int, types.Uint); err != nil {
		return ss.error(err)
	}

entryRecord:

	e.Tag = fast.BytesToString(ss.tag)
	e.Timestamp = ss.iter.Time()

	// 2) Record
	if err = ss.iter.NextExpectedType(types.Map); err != nil {
		return ss.error(err)
	}

	e.Record = ss.iter

	return
}

func (ss *ServerSession) Close() error {
	ss.resumeMessageMode()
	ss.serv.iterPool.Put(ss.iter)
	ss.serv.bufPool.Put(ss.write.Buffer)
	return ss.conn.Close()
}

func (ss *ServerSession) Rewind() {
	ss.iter.Rewind()
}

func (ss *ServerSession) prepareForMessage() {
	ss.iter.Flush()
	ss.conn.SetReadDeadline(time.Now().Add(time.Second))

	// if dur := ss.serv.opt.ReadTimeout; dur > 0 {
	// 	deadline := time.Now().Add(dur)
	// 	// dur += time.Since(ss.timeConn)
	// 	// deadline := ss.timeConn.Add(dur)
	// 	// log.Println("duration:", dur)
	// 	// log.Println("read deadline set to:", deadline)
	// 	err := ss.conn.SetReadDeadline(deadline)

	// 	_ = err
	// }
}

func (ss *ServerSession) resumeMessageMode() {
	r := ss.iter.RingReader().Reader()

	if gzip, ok := r.(*gzip.Reader); ok {
		ss.serv.gzipPool.Put(gzip)
	}

	if ss.origIter != nil {
		ss.serv.iterPool.Put(ss.iter)
		ss.iter, ss.origIter = ss.origIter, nil
	}

	ss.iter.SetManualFlush(true)
	ss.mode = MessageMode
}

// Iterate options to find "chunk" value, and write acknowledgement back to client.
func (ss *ServerSession) writeAck() (err error) {
	for range ss.iter.Items() {
		if err = ss.iter.NextExpectedType(types.Str); err != nil {
			return
		}
		key := ss.iter.Str()

		if !ss.iter.Next() {
			if err = ss.iter.Error(); err != nil {
				return
			}

			return io.ErrUnexpectedEOF
		}

		if key != "chunk" {
			ss.iter.Skip()
			continue
		}

		chunk := ss.iter.Str()
		ss.write.WriteMapHeader(1)
		ss.write.WriteString("ack")
		ss.write.WriteString(chunk)

		if _, err = ss.write.WriteTo(ss.conn); err != nil {
			return
		}
	}

	return
}

func isGzip(r *ringbuf.LimitedReader) (ok bool, err error) {
	magicNumbers, err := r.Peek(3)

	if err != nil {
		return
	}

	// Bounds check hint to compiler; see golang.org/issue/14808
	_ = magicNumbers[2]

	ok = (magicNumbers[0] == 0x1f &&
		magicNumbers[1] == 0x8b &&
		magicNumbers[2] == 8)

	return
}

func (ss *ServerSession) error(e any) (err error) {
	_, _, line, _ := runtime.Caller(1)

	if origErr, ok := e.(error); ok {
		return fmt.Errorf("line %3d: %w", line, origErr)
	}

	return fmt.Errorf("line %3d: %v", line, e)
}
