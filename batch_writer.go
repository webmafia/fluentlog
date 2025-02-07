package fluentlog

import "io"

type BatchWriter interface {
	WriteBatch(tag string, size int, r io.Reader) (err error)
}
