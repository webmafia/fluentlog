package msgpack

import (
	"io"
	"time"
)

type Writer struct {
	w   io.Writer
	buf []byte
}

// NewWriter creates a new Writer with the provided io.Writer and initial buffer size.
func NewWriter(w io.Writer, buffer []byte) Writer {
	return Writer{
		w:   w,
		buf: buffer[:0],
	}
}

func (w *Writer) Reset(wr io.Writer) {
	w.w = wr
	w.buf = w.buf[:0]
}

// WriteArrayHeader appends an array header to the buffer.
func (w *Writer) WriteArrayHeader(n int) {
	w.buf = AppendArray(w.buf, n)
}

// WriteMapHeader appends a map header to the buffer.
func (w *Writer) WriteMapHeader(n int) {
	w.buf = AppendMap(w.buf, n)
}

// WriteString appends a string to the buffer.
func (w *Writer) WriteString(s string) {
	w.buf = AppendString(w.buf, s)
}

// WriteInt appends an integer to the buffer.
func (w *Writer) WriteInt(i int64) {
	w.buf = AppendInt(w.buf, i)
}

// WriteUint appends an unsigned integer to the buffer.
func (w *Writer) WriteUint(i uint64) {
	w.buf = AppendUint(w.buf, i)
}

// WriteNil appends a nil value to the buffer.
func (w *Writer) WriteNil() {
	w.buf = AppendNil(w.buf)
}

// WriteBool appends a boolean to the buffer.
func (w *Writer) WriteBool(b bool) {
	w.buf = AppendBool(w.buf, b)
}

// WriteBinary appends binary data to the buffer.
func (w *Writer) WriteBinary(data []byte) {
	w.buf = AppendBinary(w.buf, data)
}

// WriteFloat32 appends a 32-bit floating-point number to the buffer.
func (w *Writer) WriteFloat32(f float32) {
	w.buf = AppendFloat32(w.buf, f)
}

// WriteFloat64 appends a 64-bit floating-point number to the buffer.
func (w *Writer) WriteFloat64(f float64) {
	w.buf = AppendFloat64(w.buf, f)
}

// WriteTimestamp appends a timestamp to the buffer.
func (w *Writer) WriteTimestamp(t time.Time) {
	w.buf = AppendTimestamp(w.buf, t)
}

// WriteTimestamp appends a timestamp with millisecond precision to the buffer.
func (w *Writer) WriteTimestampExt(t time.Time) {
	w.buf = AppendTimestampExt(w.buf, t)
}

// WriteCustom appends custom data to the buffer using a provided function.
func (w *Writer) WriteCustom(fn func([]byte) []byte) {
	w.buf = fn(w.buf)
}

// Flush writes the buffered data to the underlying writer and clears the buffer.
func (w *Writer) Flush() error {
	if len(w.buf) == 0 {
		return nil // Nothing to flush
	}

	_, err := w.w.Write(w.buf)
	if err != nil {
		return err
	}

	// Clear the buffer
	w.buf = w.buf[:0]
	return nil
}

// Buffer returns the current buffer without flushing it, useful for debugging or additional operations.
func (w *Writer) Buffer() []byte {
	return w.buf
}
