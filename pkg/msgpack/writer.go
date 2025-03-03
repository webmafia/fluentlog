package msgpack

import (
	"io"
	"log"

	"github.com/webmafia/fast/buffer"
)

type Writer struct {
	Buffer
	w io.Writer
}

// NewWriter creates a new Writer with the provided io.Writer and initial buffer size.
func NewWriter(w io.Writer, buf *buffer.Buffer) Writer {
	return Writer{
		w:      w,
		Buffer: Buffer{buf},
	}
}

func (w *Writer) Reset(writer io.Writer) {
	w.Buffer.Reset()
	w.w = writer
}

func (w Writer) Flush() (err error) {
	if w.w != nil {
		_, err = w.Buffer.WriteTo(w.w)
	}

	if err == nil {
		w.Buffer.Reset()
	}

	return
}

// WriteCustom appends custom data to the buffer using a provided function.
func (w Writer) WriteCustom(fn func([]byte) []byte) {
	w.Buffer.B = fn(w.Buffer.B)
}

func (w Writer) WriteBinaryReader(size int, r io.Reader) (err error) {
	log.Println("writing", size, "bytes binary")
	w.Buffer.B = appendBinaryHeader(w.Buffer.B, size)

	if err = w.Flush(); err != nil {
		return
	}

	if w.w != nil {
		if err = w.Buffer.Grow(4096); err != nil {
			return
		}

		w.Buffer.B = w.Buffer.B[:cap(w.Buffer.B)]

		if _, err = io.CopyBuffer(w.w, r, w.Buffer.B); err != nil {
			return
		}

		w.Buffer.Reset()
	}

	return
}
