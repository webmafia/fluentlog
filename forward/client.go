package forward

import (
	"net"
)

type Client struct {
	addr string
	conn net.Conn
	buf  []byte
}

func NewClient(addr string) *Client {
	return &Client{
		addr: addr,
		buf:  make([]byte, 0, 4096),
	}
}

func (c *Client) connect() (err error) {
	if c.conn, err = net.Dial("tcp", c.addr); err != nil {
		return
	}
}

func (c *Client) readHelo() (err error) {

}
