package forward

import (
	"context"
	"log"
	"net"

	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/internal/msgpack"
)

type Client struct {
	addr           string
	conn           net.Conn
	r              msgpack.Reader
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
		r:    msgpack.NewReader(nil, make([]byte, 4096)),
		w:    msgpack.NewWriter(nil, make([]byte, 4096)),
		opt:  opt,
	}
}

func (c *Client) Connect(ctx context.Context) (err error) {
	var d net.Dialer
	if c.conn, err = d.DialContext(ctx, "tcp", c.addr); err != nil {
		return
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
	c.r.Release()

	return
}

func (c *Client) Send(msg fluentlog.Message) (err error) {
	n, err := msg.WriteTo(c.conn)

	log.Println("sent", n, "bytes")
	return
}

func (c *Client) Writer() msgpack.Writer {
	return c.w
}
