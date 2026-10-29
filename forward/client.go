package forward

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

var _ io.WriteCloser = (*Client)(nil)

type Client struct {
	addr           string
	conn           net.Conn
	r              msgpack.Iterator
	w              msgpack.Writer
	opt            ClientOptions
	serverHostname string
	keepAlive      bool

	batchWriter batchWriter
	fileReader  batchFileReader
}

type ClientOptions struct {
	Hostname string
	Auth     AuthClient
	TLS      bool
}

func NewClient(addr string, opt ClientOptions) *Client {
	return &Client{
		addr: addr,
		r:    msgpack.NewIterator(nil),
		w:    msgpack.NewWriter(nil, buffer.NewBuffer(4096)),
		opt:  opt,
	}
}

func (c *Client) Connect(ctx context.Context) (err error) {
	var (
		dial net.Dialer
		tcp  *net.TCPConn
		ok   bool
		cred Credentials
	)

	if conn, err := dial.DialContext(ctx, "tcp", c.addr); err != nil {
		return errors.Join(ErrFailedConn, err)
	} else if tcp, ok = conn.(*net.TCPConn); !ok {
		return errors.New("invalid TCP connection")
	}

	tcp.SetNoDelay(true)

	// DialContext does not apply its deadline to the authentication handshake.
	deadline := time.Now().Add(3 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok {
		deadline = ctxDeadline
	}
	if err = tcp.SetDeadline(deadline); err != nil {
		tcp.Close()
		return
	}

	if c.opt.TLS {
		c.conn = tls.Client(tcp, &tls.Config{InsecureSkipVerify: true})
	} else {
		c.conn = tcp
	}

	defer func() {
		if err == nil {
			err = c.conn.SetDeadline(time.Time{})
		}
		if err != nil {
			c.Close()
		}
	}()

	if cred, err = c.opt.Auth(ctx); err != nil {
		return
	}

	c.r.Reset(c.conn)
	c.w.Reset(c.conn)

	var salt [24]byte

	if _, err = rand.Read(salt[:]); err != nil {
		return
	}

	nonce, auth, err := c.readHelo()

	if err != nil {
		return
	}

	if err = c.writePing(&cred, fast.BytesToString(salt[:]), nonce, auth); err != nil {
		return
	}

	if err = c.readPong(fast.BytesToString(salt[:]), nonce, cred.SharedKey); err != nil {
		return
	}

	return
}

func (c *Client) ensureConnection() (err error) {
	if c.conn != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Todo: Retry (https://github.com/cenkalti/backoff)
	if err = c.Connect(ctx); err != nil {
		return
	}

	return
}

func (c *Client) Write(b []byte) (n int, err error) {
	if err = c.ensureConnection(); err != nil {
		return
	}

	if err = c.conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err == nil {
		n, err = c.conn.Write(b)
	}
	if err != nil {
		c.Close()
	}
	return
}

func (c *Client) Reconnect() (err error) {
	log.Println("Reconnecting...")

	if err = c.Close(); err != nil {
		return
	}

	return c.ensureConnection()
}

// Close implements io.WriteCloser.
func (c *Client) Close() (err error) {
	if c.conn == nil {
		return nil
	}

	err = c.conn.Close()
	c.conn = nil
	return
}

func (c *Client) WriteBatch(tag string, size int, r io.Reader) (err error) {
	if err = c.ensureConnection(); err != nil {
		return
	}

	// Keep the original file visible to Linux TCP ReaderFrom instead of File.WriterTo.
	if _, file := r.(*os.File); file {
		c.fileReader.Reader = r
		r = &c.fileReader
	}
	c.batchWriter.Conn = c.conn
	c.batchWriter.size = int64(size)
	c.w.Reset(&c.batchWriter)
	defer func() {
		c.batchWriter.Conn = nil
		c.fileReader.Reader = nil
		if err != nil {
			c.Close()
		}
	}()

	c.w.WriteArrayHeader(3)

	// 1. Tag (string)
	c.w.WriteString(tag)

	// 2. Entries (CompressedMessagePackEventStream)
	if err = c.w.WriteBinaryReader(size, r); err != nil {
		return
	}

	// 3. Options
	c.w.WriteMapHeader(1)
	c.w.WriteString("compressed")
	c.w.WriteString("gzip")

	return c.w.Flush()
}

// Refresh deadlines between bounded writes, while preserving file-to-TCP
// streaming through ReaderFrom and reusing a buffer for other connections.
type batchWriter struct {
	net.Conn
	reader io.LimitedReader
	buf    []byte
	size   int64
}

func (w *batchWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		size := min(len(p), 64*1024)
		if err = w.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
			return
		}
		written, writeErr := w.Conn.Write(p[:size])
		n += written
		if writeErr != nil {
			return n, writeErr
		}
		if written != size {
			return n, io.ErrShortWrite
		}
		p = p[size:]
	}
	return
}

func (w *batchWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if source, ok := r.(*batchFileReader); ok {
		r = source.Reader
	}
	// TCP can transfer files directly. TLS and other connections need a reusable copy buffer.
	if _, tcp := w.Conn.(*net.TCPConn); !tcp && w.buf == nil {
		w.buf = make([]byte, 64*1024)
	}
	w.reader.R = r
	defer func() { w.reader.R = nil }()
	for n < w.size {
		w.reader.N = min(w.size-n, 64*1024)
		if err = w.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
			return
		}
		written, copyErr := io.CopyBuffer(w.Conn, &w.reader, w.buf)
		n += written
		if copyErr != nil || w.reader.N > 0 {
			return n, copyErr
		}
	}
	return
}

// Keep files from invoking WriterTo before the chunked transfer is selected.
type batchFileReader struct{ io.Reader }
