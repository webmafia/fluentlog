package internal

import "io"

var _ io.Reader = noop{}

type noop struct{}

// Read implements io.Reader.
func (noop) Read(_ []byte) (n int, err error) {
	return 0, io.EOF
}
