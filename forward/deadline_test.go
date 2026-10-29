package forward

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/webmafia/fluentlog/fallback"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

func TestHandshakeTimeoutAndReconnect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	release := make(chan struct{})
	defer close(release)
	served := make(chan error, 1)
	go func() {
		first, err := ln.Accept()
		if err != nil {
			served <- err
			return
		}
		defer first.Close()
		second, err := ln.Accept()
		if err != nil {
			served <- err
			return
		}
		s := NewServer(ServerOptions{Auth: StaticAuthServer(Credentials{SharedKey: "secret"})})
		ss := newServerSession(s, second)
		defer ss.Close()
		served <- ss.authenticate(context.Background())
		<-release
	}()
	c := NewClient(ln.Addr().String(), ClientOptions{Auth: StaticAuthClient(Credentials{SharedKey: "secret"})})
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := c.Connect(ctx); err == nil {
		t.Fatal("stalled HELO did not time out")
	}
	if c.conn != nil {
		t.Fatal("failed handshake retained its connection")
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}
func TestWritesAreBoundedAndResetConnection(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(map[bool]string{false: "message", true: "batch"}[batch], func(t *testing.T) {
			t.Parallel()
			a, b := net.Pipe()
			defer b.Close()
			c := NewClient("", ClientOptions{})
			c.conn = a
			c.w.Reset(a)
			var err error
			if batch {
				err = c.WriteBatch("auth", 7, strings.NewReader("payload"))
			} else {
				_, err = c.Write([]byte("payload"))
			}
			if err == nil {
				t.Fatal("stalled write did not time out")
			}
			if c.conn != nil {
				t.Fatal("failed write retained connection")
			}
		})
	}
}

func TestBatchAllowsSlowProgressBeyondThreeSeconds(t *testing.T) {
	for _, source := range []string{"bytes", "file"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()

			a, b := net.Pipe()
			defer b.Close()
			c := NewClient("", ClientOptions{})
			c.conn = a
			c.w.Reset(a)
			defer c.Close()

			payload := bytes.Repeat([]byte("x"), 256*1024)
			received := make(chan []byte, 1)
			go func() {
				var wire bytes.Buffer
				buf := make([]byte, 4096)
				for {
					n, err := b.Read(buf)
					wire.Write(buf[:n])
					if err != nil {
						received <- wire.Bytes()
						return
					}
					time.Sleep(60 * time.Millisecond)
				}
			}()

			var reader io.Reader = bytes.NewReader(payload)
			if source == "file" {
				file, err := os.CreateTemp(t.TempDir(), "batch")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				if _, err := file.Write(payload); err != nil {
					t.Fatal(err)
				}
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				reader = file
			}
			started := time.Now()
			if err := c.WriteBatch("auth", len(payload), reader); err != nil {
				t.Fatalf("steadily progressing batch: %v", err)
			}
			if time.Since(started) <= 3*time.Second {
				t.Fatal("test did not exceed the previous whole-batch deadline")
			}
			if err := c.Close(); err != nil {
				t.Fatal(err)
			}

			expected := msgpack.AppendArrayHeader(nil, 3)
			expected = msgpack.AppendString(expected, "auth")
			expected = msgpack.AppendBinary(expected, payload)
			expected = msgpack.AppendMapHeader(expected, 1)
			expected = msgpack.AppendString(expected, "compressed")
			expected = msgpack.AppendString(expected, "gzip")
			if wire := <-received; !bytes.Equal(wire, expected) {
				t.Fatalf("batch framing or payload changed: received %d bytes, expected %d", len(wire), len(expected))
			}
		})
	}
}

func TestStalledBatchRemainsBufferedForReplay(t *testing.T) {
	disk := fallback.NewDirBuffer(t.TempDir())
	defer disk.Close()
	record := bytes.Repeat([]byte("buffered record"), 1024)
	if _, err := disk.Write(record); err != nil {
		t.Fatal(err)
	}
	if has, err := disk.HasData(); err != nil || !has {
		t.Fatalf("prepare buffer: hasData=%v, err=%v", has, err)
	}

	a, b := net.Pipe()
	defer b.Close()
	c := NewClient("", ClientOptions{})
	c.conn = a
	c.w.Reset(a)
	defer c.Close()
	progress := make(chan int64, 1)
	go func() {
		// Accept part of the batch, then leave the connection open without reading.
		n, _ := io.CopyN(io.Discard, b, 32)
		progress <- n
	}()
	writeBatch := func(size int, r io.Reader) error { return c.WriteBatch("auth", size, r) }
	var timeout net.Error
	if err := disk.Reader(writeBatch); !errors.As(err, &timeout) || !timeout.Timeout() {
		t.Fatalf("expected stalled batch timeout, got %v", err)
	}
	if n := <-progress; n != 32 {
		t.Fatalf("batch did not make progress before stalling: read %d bytes", n)
	}
	if c.conn != nil {
		t.Fatal("failed batch retained its connection")
	}
	if has, err := disk.HasData(); err != nil || !has {
		t.Fatalf("failed batch was not retained: hasData=%v, err=%v", has, err)
	}

	// Retry the retained file over a healthy connection.
	a, healthy := net.Pipe()
	defer healthy.Close()
	c.conn = a
	c.w.Reset(a)
	received := make(chan []byte, 1)
	go func() { wire, _ := io.ReadAll(healthy); received <- wire }()
	if err := disk.Reader(writeBatch); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	wire := <-received
	_, offset, err := msgpack.ReadString(wire, 1)
	if err != nil {
		t.Fatal(err)
	}
	compressed, _, err := msgpack.ReadBinary(wire, offset)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	got, err := io.ReadAll(gz)
	if err != nil || !bytes.Equal(got, record) {
		t.Fatalf("replayed record changed: length=%d, err=%v", len(got), err)
	}
	if has, err := disk.HasData(); err != nil || has {
		t.Fatalf("successful batch was not removed: hasData=%v, err=%v", has, err)
	}
}

// Linux's TCP file-transfer fast path needs the original *os.File, not the
// wrapper introduced by File.WriteTo. Check the source at that dispatch point.
type fileCheckingConn struct {
	net.Conn
	file   *os.File
	copies int
}

func (c *fileCheckingConn) ReadFrom(r io.Reader) (int64, error) {
	limited, ok := r.(*io.LimitedReader)
	if !ok || limited.R != c.file {
		return 0, errors.New("batch copy lost the original file")
	}
	c.copies++
	return io.Copy(io.Discard, r)
}

func TestBatchPreservesFileForTCPReadFrom(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "batch")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	const size = 128 * 1024
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	conn := &fileCheckingConn{Conn: newMockConn(nil), file: file}
	c := NewClient("", ClientOptions{})
	c.conn = conn
	c.w.Reset(conn)
	defer c.Close()
	if err := c.WriteBatch("auth", size, file); err != nil {
		t.Fatal(err)
	}
	if conn.copies == 0 {
		t.Fatal("batch bypassed ReaderFrom")
	}
}
