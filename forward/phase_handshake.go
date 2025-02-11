package forward

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/webmafia/fluentlog/internal"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
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

	// 0) Array of length 2
	if err = c.r.NextExpectedType(types.Array); err != nil {
		return nil, errors.Join(ErrInvalidHelo, err)
	}
	if c.r.Items() != 2 {
		return nil, ErrInvalidHelo
	}

	// 1) Type
	if err = c.r.NextExpectedType(types.Str); err != nil {
		return nil, errors.Join(ErrInvalidHelo, err)
	}
	if typ := c.r.Str(); typ != HELO {
		return nil, fmt.Errorf("%w: expected type '%s', got '%s'", ErrInvalidHelo, HELO, typ)
	}

	// 2) Options
	if err = c.r.NextExpectedType(types.Map); err != nil {
		return nil, errors.Join(ErrInvalidHelo, err)
	}

	var (
		authSalt  []byte
		keepAlive = true
	)

	for range c.r.Items() {
		if err = c.r.NextExpectedType(types.Str); err != nil {
			return nil, errors.Join(ErrInvalidHelo, err)
		}

		key := c.r.Str()

		if !c.r.Next() {
			return nil, c.r.Error()
		}

		switch key {

		case "nonce":
			nonce = c.r.Bin()

		case "auth":
			authSalt = c.r.Bin()

		case "keepalive":
			keepAlive = c.r.Bool()

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
	var clientHostname, digest string

	// 0) Array of 6 items
	if err = s.r.NextExpectedType(types.Array); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	if s.r.Items() != 6 {
		return nil, nil, ErrInvalidPing
	}

	// 1) Type
	if err = s.r.NextExpectedType(types.Str); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	if typ := s.r.Str(); typ != PING {
		return nil, nil, fmt.Errorf("%w: expected type '%s', got '%s'", ErrInvalidPing, PING, typ)
	}

	// 2) Client hostname
	if err = s.r.NextExpectedType(types.Str); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	clientHostname = s.r.Str()

	// 3) Shared key salt
	if err = s.r.NextExpectedType(types.Str, types.Bin); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	salt = s.r.Bin()

	// 4) Shared key hexdigest
	if err = s.r.NextExpectedType(types.Str); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	digest = s.r.Str()

	// 5) Username
	if err = s.r.NextExpectedType(types.Str); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	_ = s.r.Str()

	// 5) Password
	if err = s.r.NextExpectedType(types.Str); err != nil {
		return nil, nil, errors.Join(ErrInvalidPing, err)
	}
	_ = s.r.Str()

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

	// 0) Array of 5 items
	if err = c.r.NextExpectedType(types.Array); err != nil {
		return errors.Join(ErrInvalidPong, err)
	}
	if c.r.Items() != 5 {
		return ErrInvalidPong
	}

	// 1) Type
	if err = c.r.NextExpectedType(types.Str); err != nil {
		return errors.Join(ErrInvalidPong, err)
	}
	if typ := c.r.Str(); typ != PONG {
		return fmt.Errorf("%w: expected type '%s', got '%s'", ErrInvalidPong, PONG, typ)
	}

	// 2) Auth result
	if err = c.r.NextExpectedType(types.Bool); err != nil {
		return errors.Join(ErrInvalidPong, err)
	}
	authResult := c.r.Bool()

	// 3) Reason
	if err = c.r.NextExpectedType(types.Str); err != nil {
		return errors.Join(ErrInvalidPong, err)
	}
	reason := c.r.Str()

	// 4) Server hostname
	if err = c.r.NextExpectedType(types.Str); err != nil {
		return errors.Join(ErrInvalidPong, err)
	}
	serverHostname := c.r.Str()

	// 5) Shared key hexdigest
	if err = c.r.NextExpectedType(types.Str); err != nil {
		return errors.Join(ErrInvalidPong, err)
	}
	digest := c.r.Str()

	if !authResult {
		if reason != "" {
			return fmt.Errorf("%w: %s", ErrFailedAuth, reason)
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

func passwordDigest(salt []byte, username, password string) string {
	h := sha512.New()
	h.Write(salt)
	h.Write(internal.S2B(username))
	h.Write(internal.S2B(password))

	return internal.B2S(hex.AppendEncode(nil, h.Sum(nil)))
}
