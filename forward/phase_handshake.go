package forward

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/webmafia/fluentlog/internal"
)

const (
	HELO = "HELO"
	PING = "PING"
	PONG = "PONG"
)

func (s *ServerConn) writeHelo() (nonce [16]byte, err error) {
	if _, err = rand.Read(nonce[:]); err != nil {
		return
	}

	s.w.WriteArrayHeader(2)
	s.w.WriteString(HELO)
	s.w.WriteMapHeader(3)

	s.w.WriteString("nonce")
	s.w.WriteBinary(nonce[:])

	s.w.WriteString("auth")
	s.w.WriteBinary(nil)

	s.w.WriteString("keepalive")
	s.w.WriteBool(true)

	return nonce, s.w.Flush()
}

func (c *Client) readHelo() (nonce []byte, err error) {
	arrLen, err := c.r.ReadArrayHeader()

	if err != nil {
		return
	}

	if arrLen != 2 {
		return nil, ErrInvalidHelo
	}

	typ, err := c.r.ReadString()

	if err != nil {
		return
	}

	if typ != HELO {
		return nil, ErrInvalidHelo
	}

	mapLen, err := c.r.ReadMapHeader()

	if err != nil {
		return
	}

	var (
		key       string
		authSalt  []byte
		keepAlive = true
	)

	for range mapLen {
		if key, err = c.r.ReadString(); err != nil {
			return
		}

		switch key {

		case "nonce":
			if nonce, err = c.r.ReadBinary(); err != nil {
				return
			}

		case "auth":
			if authSalt, err = c.r.ReadBinary(); err != nil {
				return
			}

		case "keepalive":
			if keepAlive, err = c.r.ReadBool(); err != nil {
				return
			}

		}
	}

	log.Println("nonce:", nonce)
	log.Println("auth:", authSalt)
	log.Println("keepalive:", keepAlive)

	if len(authSalt) > 0 {
		// log.Println("auth salt:", string(authSalt))
		// return nil, errors.New("server requires auth, which isn't yet supported in client")
	}

	if len(nonce) == 0 {
		return nil, ErrInvalidNonce
	}

	c.keepAlive = keepAlive
	return
}

func (c *Client) writePing(nonce []byte) (salt [16]byte, err error) {
	if _, err = rand.Read(salt[:]); err != nil {
		return
	}

	c.w.WriteArrayHeader(6)

	c.w.WriteString(PING)
	c.w.WriteString(c.opt.Hostname)
	c.w.WriteString(internal.B2S(salt[:]))
	c.w.WriteString(sharedKeyDigest(salt[:], c.opt.Hostname, nonce, c.opt.SharedKey))
	c.w.WriteString("")
	c.w.WriteString("")

	if err = c.w.Flush(); err != nil {
		return
	}

	return
}

func (s *ServerConn) readPing(nonce []byte) (salt []byte, sharedKey []byte, err error) {
	arrLen, err := s.r.ReadArrayHeader()

	if err != nil {
		return
	}

	if arrLen != 6 {
		return nil, nil, ErrInvalidPing
	}

	typ, err := s.r.ReadString()

	if err != nil {
		return
	}

	if typ != PING {
		return nil, nil, ErrInvalidPing
	}

	clientHostname, err := s.r.ReadString()

	if err != nil {
		return
	}

	if salt, err = s.r.ReadBinary(); err != nil {
		return
	}

	digest, err := s.r.ReadString()

	if err != nil {
		return
	}

	// Skip username
	if _, err = s.r.ReadString(); err != nil {
		return
	}

	// Skip password
	if _, err = s.r.ReadString(); err != nil {
		return
	}

	if sharedKey, err = s.serv.opt.SharedKey(clientHostname); err != nil {
		return
	}

	correctDigest := sharedKeyDigest(salt, clientHostname, nonce, sharedKey)

	if !internal.SameString(digest, correctDigest) {
		return nil, nil, ErrInvalidSharedKey
	}

	return
}

func (s *ServerConn) writePong(nonce []byte, salt []byte, sharedKey []byte, authResult bool, reason string) (err error) {
	s.w.WriteArrayHeader(5)
	s.w.WriteString(PONG)
	s.w.WriteBool(authResult)
	s.w.WriteString(reason)
	s.w.WriteString(s.serv.opt.Hostname)
	s.w.WriteString(sharedKeyDigest(salt, s.serv.opt.Hostname, nonce, sharedKey))
	return s.w.Flush()
}

func (c *Client) readPong(nonce []byte, salt [16]byte) (err error) {
	arrLen, err := c.r.ReadArrayHeader()

	if err != nil {
		return
	}

	if arrLen != 5 {
		return ErrInvalidPong
	}

	typ, err := c.r.ReadString()

	if err != nil {
		return
	}

	if typ != PONG {
		return ErrInvalidPong
	}

	authResult, err := c.r.ReadBool()

	if err != nil {
		return
	}

	reason, err := c.r.ReadString()

	if err != nil {
		return
	}

	serverHostname, err := c.r.ReadString()

	if err != nil {
		return
	}

	digest, err := c.r.ReadString()

	if err != nil {
		return
	}

	if !authResult {
		if reason != "" {
			return fmt.Errorf("%w - reason: %s", ErrFailedAuth, reason)
		}

		return ErrFailedAuth
	}

	correctDigest := sharedKeyDigest(salt[:], serverHostname, nonce, c.opt.SharedKey)

	if !internal.SameString(digest, correctDigest) {
		return ErrFailedAuth
	}

	c.serverHostname = strings.Clone(serverHostname)
	return
}

func sharedKeyDigest(salt []byte, fqdn string, nonce []byte, sharedKey []byte) string {
	h := sha512.New()
	h.Write(salt)
	h.Write(internal.S2B(fqdn))
	h.Write(nonce)
	h.Write(sharedKey)

	return internal.B2S(hex.AppendEncode(nil, h.Sum(nil)))
}
