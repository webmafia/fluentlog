package internal

import (
	"crypto/subtle"
	"unsafe"
)

//go:inline
func S2B(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

//go:inline
func B2S(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Constant-time string comparison
func SameString(a, b string) bool {
	return subtle.ConstantTimeCompare(S2B(a), S2B(b)) == 1
}
