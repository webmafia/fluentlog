package forward

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

const (
	HELO = "HELO"
	PING = "PING"
	PONG = "PONG"
)

func (s *ServerConn) writeHelo(nonce, auth string) error {
	s.w.WriteArrayHeader(2)
	s.w.WriteString(HELO)
	s.w.WriteMapHeader(3)

	s.w.WriteString("nonce")
	s.w.WriteString(nonce)

	s.w.WriteString("auth")
	s.w.WriteString(auth)

	s.w.WriteString("keepalive")
	s.w.WriteBool(true)

	return s.w.Flush()
}

func (c *Client) readHelo() (nonce, auth string, err error) {

	// 0) Array of length 2
	if err = c.r.NextExpectedType(types.Array); err != nil {
		return "", "", errors.Join(ErrInvalidHelo, err)
	}
	if c.r.Items() != 2 {
		return "", "", ErrInvalidHelo
	}

	// 1) Type
	if err = c.r.NextExpectedType(types.Str); err != nil {
		return "", "", errors.Join(ErrInvalidHelo, err)
	}
	if typ := c.r.Str(); typ != HELO {
		return "", "", fmt.Errorf("%w: expected type '%s', got '%s'", ErrInvalidHelo, HELO, typ)
	}

	// 2) Options
	if err = c.r.NextExpectedType(types.Map); err != nil {
		return "", "", errors.Join(ErrInvalidHelo, err)
	}

	var keepAlive = true

	for range c.r.Items() {
		if err = c.r.NextExpectedType(types.Str); err != nil {
			return "", "", errors.Join(ErrInvalidHelo, err)
		}

		key := c.r.Str()

		if !c.r.Next() {
			return "", "", c.r.Error()
		}

		switch key {

		case "nonce":
			nonce = c.r.Str()

		case "auth":
			auth = c.r.Str()

		case "keepalive":
			keepAlive = c.r.Bool()

		}
	}

	if len(nonce) == 0 {
		return "", "", ErrInvalidNonce
	}

	c.keepAlive = keepAlive
	return
}

func (c *Client) writePing(cred *Credentials, salt, nonce, auth string) (err error) {
	c.w.WriteArrayHeader(6)

	c.w.WriteString(PING)
	c.w.WriteString(c.opt.Hostname)
	c.w.WriteString(salt)
	c.w.WriteStringMax255(sha512Hex(salt, c.opt.Hostname, nonce, cred.SharedKey))

	if auth != "" {
		c.w.WriteString(cred.Username)
		c.w.WriteStringMax255(sha512Hex(auth, cred.Username, cred.Password))
	} else {
		c.w.WriteString("")
		c.w.WriteString("")
	}

	return c.w.Flush()
}

func (s *ServerConn) readPing(ctx context.Context, nonce, auth string) (salt string, cred Credentials, err error) {
	var clientHostname string

	// 0) Array of 6 items
	if err = s.r.NextExpectedType(types.Array); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	if s.r.Items() != 6 {
		err = ErrInvalidPing
		return
	}

	// 1) Type
	if err = s.r.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	if typ := s.r.Str(); typ != PING {
		err = fmt.Errorf("%w: expected type '%s', got '%s'", ErrInvalidPing, PING, typ)
		return
	}

	// 2) Client hostname
	if err = s.r.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	clientHostname = s.r.Str()

	// 3) Shared key salt
	if err = s.r.NextExpectedType(types.Str, types.Bin); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	salt = s.r.Str()

	var (
		hexdigest []byte
		username  string
		password  []byte
	)

	// 4) Shared key hexdigest
	if err = s.r.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	hexdigest = s.r.Bin()

	// 5) Username
	if err = s.r.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	username = s.r.Str()

	// 5) Password
	if err = s.r.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	password = s.r.Bin()

	if cred, err = s.serv.opt.Auth(ctx, username); err != nil {
		err = errors.Join(ErrFailedAuth, err)
		return
	}

	// Validate shared key
	if !validateSha512Hex(hexdigest, salt, clientHostname, nonce, cred.SharedKey) {
		err = ErrInvalidSharedKey
		return
	}

	// Validate password
	if auth != "" {
		if !validateSha512Hex(password, auth, username, cred.Password) {
			err = ErrFailedAuth
			return
		}
	}

	return
}

func (s *ServerConn) writePong(salt string, nonce string, sharedKey string, authResult bool, reason string) (err error) {
	s.w.WriteArrayHeader(5)
	s.w.WriteString(PONG)
	s.w.WriteBool(authResult)
	s.w.WriteString(reason)
	s.w.WriteString(s.serv.opt.Hostname)
	s.w.WriteStringMax255(sha512Hex(salt, s.serv.opt.Hostname, nonce, sharedKey))
	return s.w.Flush()
}

func (c *Client) readPong(salt, nonce, sharedKey string) (err error) {

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
	hexdigest := c.r.Bin()

	if !authResult {
		if reason != "" {
			return fmt.Errorf("%w: %s", ErrFailedAuth, reason)
		}

		return ErrFailedAuth
	}

	if !validateSha512Hex(hexdigest, salt, serverHostname, nonce, sharedKey) {
		return ErrFailedAuth
	}

	c.serverHostname = strings.Clone(serverHostname)
	return
}
