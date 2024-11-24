package internal

import "unsafe"

//go:inline
func S2B(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

//go:inline
func B2S(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
