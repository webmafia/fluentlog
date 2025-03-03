package msgpack

import (
	"time"

	"github.com/webmafia/fast/buffer"
)

type Buffer struct {
	*buffer.Buffer
}

// WriteArrayHeader appends an array header to the buffer.
func (w Buffer) WriteArrayHeader(n int) {
	w.Buffer.B = AppendArrayHeader(w.Buffer.B, n)
}

// WriteMapHeader appends a map header to the buffer.
func (w Buffer) WriteMapHeader(n int) {
	w.Buffer.B = AppendMapHeader(w.Buffer.B, n)
}

// WriteString appends a string to the buffer.
func (w Buffer) WriteString(s string) {
	w.Buffer.B = AppendString(w.Buffer.B, s)
}

// WriteStringMax255 appends a string of unknown length (but
// max 255 characters) to the buffer.
func (w Buffer) WriteStringMax255(fn func(dst []byte) []byte) {
	w.Buffer.B = AppendStringMax255(w.Buffer.B, fn)
}

// WriteInt appends an integer to the buffer.
func (w Buffer) WriteInt(i int64) {
	w.Buffer.B = AppendInt(w.Buffer.B, i)
}

// WriteUint appends an unsigned integer to the buffer.
func (w Buffer) WriteUint(i uint64) {
	w.Buffer.B = AppendUint(w.Buffer.B, i)
}

// WriteNil appends a nil value to the buffer.
func (w Buffer) WriteNil() {
	w.Buffer.B = AppendNil(w.Buffer.B)
}

// WriteBool appends a boolean to the buffer.
func (w Buffer) WriteBool(b bool) {
	w.Buffer.B = AppendBool(w.Buffer.B, b)
}

// WriteBinary appends binary data to the buffer.
func (w Buffer) WriteBinary(data []byte) {
	w.Buffer.B = AppendBinary(w.Buffer.B, data)
}

// WriteFloat64 appends a 64-bit floating-point number to the buffer.
func (w Buffer) WriteFloat(f float64) {
	w.Buffer.B = AppendFloat(w.Buffer.B, f)
}

// WriteTimestamp appends a timestamp to the buffer.
func (w Buffer) WriteTimestamp(t time.Time, format ...TsFormat) {
	w.Buffer.B = AppendTimestamp(w.Buffer.B, t, format...)
}
