package forward

import (
	"crypto/rand"

	"github.com/webmafia/fast"
)

type randSalt struct {
	b     [23]byte
	isset bool
}

func newRandSalt() (s randSalt) {

	// Since Go 1.24, this will always return a nil error
	rand.Read(s.b[:])
	s.isset = true
	return
}

func (s randSalt) String() string {
	return fast.BytesToString(s.b[:])
}
