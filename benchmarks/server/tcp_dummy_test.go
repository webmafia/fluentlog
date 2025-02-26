package main

import (
	"bytes"
	"net"
	"time"
)

type dummyConn struct {
	addr dummyAddr
	data *bytes.Reader
}

func (d *dummyConn) Read(b []byte) (int, error)         { return d.data.Read(b) }
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

func tcpDummyConn(data []byte, fn func(conn *dummyConn) error) (err error) {
	conn := &dummyConn{
		data: bytes.NewReader(data),
	}

	return fn(conn)
}
