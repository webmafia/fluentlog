package main

import (
	"net"
)

func persistentTCPServer(data []byte) (ln net.Listener, err error) {
	if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
		return
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Continuously write data.
		for {
			_, err := conn.Write(data)
			if err != nil {
				return
			}
		}
	}()

	return
}

func tcpConn(data []byte, fn func(c net.Conn) error) (err error) {
	ln, err := persistentTCPServer(data)

	if err != nil {
		return
	}

	defer ln.Close()

	// Dial the connection once, outside the benchmark loop.
	conn, err := net.Dial("tcp", ln.Addr().String())

	if err != nil {
		return
	}

	defer conn.Close()

	return fn(conn)
}
