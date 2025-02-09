package forward

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Client struct {
	addr           string
	conn           *net.TCPConn
	r              msgpack.Iterator
	w              msgpack.Writer
	opt            ClientOptions
	serverHostname string
	keepAlive      bool
}

type ClientOptions struct {
	Hostname  string
	SharedKey []byte
}

func NewClient(addr string, opt ClientOptions) *Client {
	return &Client{
		addr: addr,
		r:    msgpack.NewIterator(nil, 4096),
		w:    msgpack.NewWriter(nil, buffer.NewBuffer(4096)),
		opt:  opt,
	}
}

func (c *Client) Connect(ctx context.Context) (err error) {
	var (
		dial net.Dialer
		conn net.Conn
		ok   bool
	)

	if conn, err = dial.DialContext(ctx, "tcp", c.addr); err != nil {
		return errors.Join(ErrFailedConn, err)
	}

	if c.conn, ok = conn.(*net.TCPConn); !ok {
		return ErrFailedConn
	}

	c.r.Reset(c.conn)
	c.w.Reset(c.conn)

	nonce, err := c.readHelo()

	if err != nil {
		return
	}

	salt, err := c.writePing(nonce)

	if err != nil {
		return
	}

	if err = c.readPong(nonce, salt); err != nil {
		return
	}

	log.Println("connected!")
	// c.r.Release(true)

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

	return c.conn.Write(b)
}

func (c *Client) WriteBatch(tag string, size int, r io.Reader) (err error) {
	if err = c.ensureConnection(); err != nil {
		return
	}

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
