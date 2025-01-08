package forward

import (
	"context"
	"errors"
	"log"
	"net"
)

func udpConn(ctx context.Context, addr string) (conn *net.UDPConn, err error) {
	var lc net.ListenConfig
	c, err := lc.ListenPacket(ctx, "udp", addr)

	if err != nil {
		return
	}

	conn, ok := c.(*net.UDPConn)

	if !ok {
		return nil, errors.New("invalid UDP connection")
	}

	return
}

func (s *Server) listenHeartbeat(ctx context.Context) (conn *net.UDPConn, err error) {
	if conn, err = udpConn(ctx, s.opt.Address); err != nil {
		return
	}

	go func() {
		log.Println("Listening for heartbeats (UDP) on", s.opt.Address)

		var buf [1]byte

		for {
			n, addr, err := conn.ReadFromUDPAddrPort(buf[:])

			if n > 0 {
				log.Println("Heartbeat from", addr)
				conn.WriteToUDPAddrPort(buf[:], addr)
			}

			if err != nil {
				return
			}

		}
	}()

	return
}
