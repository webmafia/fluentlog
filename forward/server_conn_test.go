package forward

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/webmafia/fluentlog/pkg/msgpack"
)

// --- Dummy implementations to satisfy dependencies ---

// dummyConn implements net.Conn for our benchmark.
type dummyConn struct {
	addr dummyAddr
}

func (d *dummyConn) Read(b []byte) (int, error)         { return 0, io.EOF }
func (d *dummyConn) Write(b []byte) (int, error)        { return len(b), nil }
func (d *dummyConn) Close() error                       { return nil }
func (d *dummyConn) LocalAddr() net.Addr                { return d.addr }
func (d *dummyConn) RemoteAddr() net.Addr               { return d.addr }
func (d *dummyConn) SetDeadline(t time.Time) error      { return nil }
func (d *dummyConn) SetReadDeadline(t time.Time) error  { return nil }
func (d *dummyConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return string(a) }
func (a dummyAddr) String() string  { return string(a) }

// --- End dummy implementations ---

// For our benchmark, we need a valid msgpack payload.
// In this example we use a Message Mode event:
//
//	[ "tag", 1441588984, {"message": "test"} ]
func createPayload() (b []byte) {
	b = msgpack.AppendArrayHeader(b, 3)
	b = msgpack.AppendString(b, "tag")
	b = msgpack.AppendInt(b, 1441588984)
	b = msgpack.AppendMapHeader(b, 1)
	b = msgpack.AppendString(b, "message")
	b = msgpack.AppendString(b, "test")

	return
}

func BenchmarkEntries(b *testing.B) {
	validPayload := createPayload()

	// Create an iterator from the valid payload.
	// We assume msgpack.NewIterator takes a byte slice.
	// it := msgpack.NewIterator(nil)
	// it.ResetBytes(validPayload)

	// Create a dummy connection.
	conn := &dummyConn{}

	s := NewServer(ServerOptions{})

	iter := s.iterPool.Get(conn)
	iter.ResetBytes(validPayload)
	wBuf := s.bufPool.Get()
	state := s.bufPool.Get()

	// Create our ServerConn.
	sc := ServerConn{
		serv:  s,
		conn:  conn,
		r:     iter,
		w:     msgpack.NewWriter(conn, wBuf),
		state: state,
	}

	// Report allocations.
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, rec := range sc.Entries() {
			rec.Skip()
		}

		// Reset the iterator for the next iteration.
		iter.ResetBytes(validPayload)
		wBuf.Reset()
		state.Reset()
	}

	b.StopTimer()
}
