package forward

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

var _ io.Writer = (*Client)(nil)

type Client struct {
    addr           string
    conn           net.Conn
    r              msgpack.Iterator
    w              msgpack.Writer
    opt            ClientOptions
    serverHostname string
    keepAlive      bool
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
        conn net.Conn
        cred Credentials
    )

    if c.opt.TLS {
        // Use system trust store; enable SNI from host in address.
        if conn, err = tls.DialWithDialer(&dial, "tcp", c.addr, &tls.Config{}); err != nil {
            return errors.Join(ErrFailedConn, err)
        }
    } else {
        if conn, err = dial.DialContext(ctx, "tcp", c.addr); err != nil {
            return errors.Join(ErrFailedConn, err)
        }
    }

    c.conn = conn

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
