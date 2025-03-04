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

func (ss *ServerSession) writeHelo(nonce, auth string) (err error) {
	ss.write.WriteArrayHeader(2)
	ss.write.WriteString(HELO)
	ss.write.WriteMapHeader(3)

	ss.write.WriteString("nonce")
	ss.write.WriteString(nonce)

	ss.write.WriteString("auth")
	ss.write.WriteString(auth)

	ss.write.WriteString("keepalive")
	ss.write.WriteBool(true)

	_, err = ss.write.WriteTo(ss.conn)
	return
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

func (ss *ServerSession) readPing(ctx context.Context, nonce, auth string) (salt string, cred Credentials, err error) {
	var clientHostname string

	// 0) Array of 6 items
	if err = ss.iter.NextExpectedType(types.Array); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	if ss.iter.Items() != 6 {
		err = ErrInvalidPing
		return
	}

	// 1) Type
	if err = ss.iter.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	if typ := ss.iter.Str(); typ != PING {
		err = fmt.Errorf("%w: expected type '%s', got '%s'", ErrInvalidPing, PING, typ)
		return
	}

	// 2) Client hostname
	if err = ss.iter.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	clientHostname = ss.iter.Str()

	// 3) Shared key salt
	if err = ss.iter.NextExpectedType(types.Str, types.Bin); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	salt = ss.iter.Str()

	var (
		hexdigest []byte
		username  string
		password  []byte
	)

	// 4) Shared key hexdigest
	if err = ss.iter.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	hexdigest = ss.iter.Bin()

	// 5) Username
	if err = ss.iter.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	username = ss.iter.Str()

	// 5) Password
	if err = ss.iter.NextExpectedType(types.Str); err != nil {
		err = errors.Join(ErrInvalidPing, err)
		return
	}
	password = ss.iter.Bin()

	if cred, err = ss.serv.opt.Auth(ctx, username); err != nil {
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

func (ss *ServerSession) writePong(salt string, nonce string, sharedKey string, authResult bool, reason string) (err error) {
	ss.write.WriteArrayHeader(5)
	ss.write.WriteString(PONG)
	ss.write.WriteBool(authResult)
	ss.write.WriteString(reason)
	ss.write.WriteString(ss.serv.opt.Hostname)
	ss.write.WriteStringMax255(sha512Hex(salt, ss.serv.opt.Hostname, nonce, sharedKey))

	_, err = ss.write.WriteTo(ss.conn)
	return
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
