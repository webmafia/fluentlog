package forward

var _ error = Error("")

type Error string

func (err Error) Error() string {
	return string(err)
}

const (
	ErrInvalidHelo      = Error("invalid HELO")
	ErrInvalidPing      = Error("invalid PING")
	ErrInvalidPong      = Error("invalid PONG")
	ErrInvalidNonce     = Error("invalid nonce")
	ErrInvalidSharedKey = Error("invalid shared key")
	ErrInvalidEntry     = Error("invalid entry")
	ErrFailedConn       = Error("failed connection")
	ErrFailedAuth       = Error("failed authentication")
)
