package fallback

import "io"

type Fallback interface {
	io.WriteCloser
	HasData() (ok bool, err error)
	Reader(fn func(n int, r io.Reader) error) (err error)
}
