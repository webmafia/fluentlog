package msgpack

import (
	"io"
	"time"

	"github.com/webmafia/fast/buffer"
)

type Writer struct {
	b *buffer.Buffer
	w io.Writer
}

// NewWriter creates a new Writer with the provided io.Writer and initial buffer size.
func NewWriter(w io.Writer, buf *buffer.Buffer) Writer {
	return Writer{
		w: w,
		b: buf,
	}
}

func (w Writer) Reset() {
	w.b.Reset()
}

func (w Writer) Bytes() []byte {
	return w.b.Bytes()
}

func (w Writer) Flush() (err error) {
	_, err = w.w.Write(w.b.B)

	if err == nil {
		w.b.Reset()
	}

	return
}

func (w Writer) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

func (w Writer) WriteTo(wr io.Writer) (int64, error) {
	return w.b.WriteTo(wr)
}

// WriteArrayHeader appends an array header to the buffer.
func (w Writer) WriteArrayHeader(n int) {
	w.b.B = AppendArrayHeader(w.b.B, n)
}

// WriteMapHeader appends a map header to the buffer.
func (w Writer) WriteMapHeader(n int) {
	w.b.B = AppendMapHeader(w.b.B, n)
}

// WriteString appends a string to the buffer.
func (w Writer) WriteString(s string) {
	w.b.B = AppendString(w.b.B, s)
}

// WriteInt appends an integer to the buffer.
func (w Writer) WriteInt(i int64) {
	w.b.B = AppendInt(w.b.B, i)
}

// WriteUint appends an unsigned integer to the buffer.
func (w Writer) WriteUint(i uint64) {
	w.b.B = AppendUint(w.b.B, i)
}

// WriteNil appends a nil value to the buffer.
func (w Writer) WriteNil() {
	w.b.B = AppendNil(w.b.B)
}

// WriteBool appends a boolean to the buffer.
func (w Writer) WriteBool(b bool) {
	w.b.B = AppendBool(w.b.B, b)
}

// WriteBinary appends binary data to the buffer.
func (w Writer) WriteBinary(data []byte) {
	w.b.B = AppendBinary(w.b.B, data)
}

// WriteFloat64 appends a 64-bit floating-point number to the buffer.
func (w Writer) WriteFloat(f float64) {
	w.b.B = AppendFloat(w.b.B, f)
}

// WriteTimestamp appends a timestamp to the buffer.
func (w Writer) WriteTimestamp(t time.Time, format ...TsFormat) {
	w.b.B = AppendTimestamp(w.b.B, t, format...)
}

// WriteCustom appends custom data to the buffer using a provided function.
func (w Writer) WriteCustom(fn func([]byte) []byte) {
	w.b.B = fn(w.b.B)
}
