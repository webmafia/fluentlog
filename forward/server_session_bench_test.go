package forward

import (
	"bytes"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/webmafia/fluentlog/forward/transport"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

// mockConn is a trivial in-memory net.Conn that returns
// data from a bytes.Reader. Good enough for benchmarking reads.
type mockConn struct {
	buf    *bytes.Reader
	closed bool
	mu     sync.Mutex
}

func newMockConn(data []byte) net.Conn {
	return &mockConn{
		buf: bytes.NewReader(data),
	}
}

func (m *mockConn) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, net.ErrClosed
	}
	return m.buf.Read(p)
}

// The rest of these methods are just stubs to satisfy net.Conn.

func (m *mockConn) Write(p []byte) (int, error) { return len(p), nil }
func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
func (m *mockConn) LocalAddr() net.Addr                { return dummyAddr("local") }
func (m *mockConn) RemoteAddr() net.Addr               { return dummyAddr("remote") }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr string

func (d dummyAddr) Network() string { return string(d) }
func (d dummyAddr) String() string  { return string(d) }

// makeMockForwardData returns a chunk of data representing `count` ForwardMode events.
// This is a simplified example using a small, fixed tag and record for each sub-event.
func makeMockForwardData(count int) []byte {
	ts := time.Now()

	var msg []byte
	msg = msgpack.AppendArrayHeader(msg, 2)
	msg = msgpack.AppendTimestamp(msg, ts)
	msg = msgpack.AppendMapHeader(msg, 1)
	msg = msgpack.AppendString(msg, "key")
	msg = msgpack.AppendString(msg, "value")

	var buf []byte
	buf = msgpack.AppendArrayHeader(buf, 2)
	buf = msgpack.AppendString(buf, "tag")
	buf = msgpack.AppendArrayHeader(buf, count)

	buf = slices.Grow(buf, len(msg)*count)

	// Insert `count` sub-events
	for range count {
		buf = append(buf, msg...)
	}

	// buf = msgpack.AppendMapHeader(buf, 0)

	return buf
}

// BenchmarkServerSessionNext measures how quickly we can call Next() on
// a ServerSession loaded with Forward-mode data.
func BenchmarkServerSessionNext(b *testing.B) {
	// b.N = 2
	// 1) Build mock data with b.N sub-events
	data := makeMockForwardData(b.N)

	// 2) Wrap it in a mock conn
	conn := newMockConn(data)

	// 3) Create a minimal server & session
	srv := &Server{}
	ss := newServerSession(srv, conn)
	ss.initTransportPhase() // e.g. sets mode = MessageMode, or however you do it

	b.ResetTimer()

	// 4) We'll read b.N sub-events, ignoring them but measuring parse overhead
	for i := 0; i < b.N; i++ {
		var e transport.Entry
		if err := ss.Next(&e); err != nil {
			b.Fatalf("Next error at i=%d: %v", i, err)
		}

		e.Record.Skip()
	}

	b.StopTimer()
}
