package internal

// These interfaces will be added to the `encoding` standard library in Go 1.24.
// Reference: https://tip.golang.org/doc/go1.24

// TextAppender is the interface implemented by an object
// that can append the textual representation of itself.
// If a type implements both [TextAppender] and [TextMarshaler],
// then v.MarshalText() must be semantically identical to v.AppendText(nil).
type TextAppender interface {
	// AppendText appends the textual representation of itself to the end of b
	// (allocating a larger slice if necessary) and returns the updated slice.
	//
	// Implementations must not retain b, nor mutate any bytes within b[:len(b)].
	AppendText(b []byte) ([]byte, error)
}

// BinaryAppender is the interface implemented by an object
// that can append the binary representation of itself.
// If a type implements both [BinaryAppender] and [BinaryMarshaler],
// then v.MarshalBinary() must be semantically identical to v.AppendBinary(nil).
type BinaryAppender interface {
	// AppendText appends the binary representation of itself to the end of b
	// (allocating a larger slice if necessary) and returns the updated slice.
	//
	// Implementations must not retain b, nor mutate any bytes within b[:len(b)].
	AppendBinary([]byte) ([]byte, error)
}
