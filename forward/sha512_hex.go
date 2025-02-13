package forward

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"hash"
	"sync"

	"github.com/webmafia/fast"
)

var sha512Pool sync.Pool

func getSha512() hash.Hash {
	if h, ok := sha512Pool.Get().(hash.Hash); ok {
		h.Reset()
		return h
	}

	return sha512.New()
}

func sha512Hex(vals ...string) func(dst []byte) []byte {
	return func(dst []byte) []byte {
		h := getSha512()
		defer sha512Pool.Put(h)

		for i := range vals {
			h.Write(fast.StringToBytes(vals[i]))
		}

		var digest [64]byte

		return hex.AppendEncode(dst, h.Sum(fast.NoescapeBytes(digest[:0])))
	}
}

func validateSha512Hex(expectedHex []byte, vals ...string) bool {
	var calcHex [128]byte
	calcHexDigest := sha512Hex(vals...)(calcHex[:0])

	return subtle.ConstantTimeCompare(expectedHex, calcHexDigest) == 1
}
