package forward

import (
	"io"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/forward/transport"
	"github.com/webmafia/fluentlog/internal/gzip"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

var _ io.Writer = (*AsciiFormatter)(nil)

type AsciiFormatter struct {
	w        io.Writer
	iter     msgpack.Iterator
	trans    transport.TransportPhase
	iterPool msgpack.IterPool
	gzipPool gzip.Pool
	buf      []byte
}

// Formats log messages from Fluent Forward protocol into ASCII.
// Handy for debugging - do not use in production.
func NewAsciiFormatter(w io.Writer) *AsciiFormatter {
	a := &AsciiFormatter{w: w, iter: msgpack.NewIterator(nil)}
	a.trans.Init(&a.iterPool, &a.gzipPool, func(_ string) error { return nil })
	return a
}

// Write implements io.Writer.
func (a *AsciiFormatter) Write(p []byte) (n int, err error) {
	a.iter.ResetBytes(p)

	var e transport.Entry

	for {
		if err = a.trans.Next(&a.iter, &e); err != nil {
			if err == io.EOF {
				return len(p), nil
			}
			return 0, err
		}

		b := a.buf[:0]

		// time
		b = e.Timestamp.AppendFormat(b, "2006-01-02 15:04:05 MST")
		b = append(b, ' ')

		// tag
		b = append(b, fast.StringToBytes(e.Tag)...)

		// record fields
		items := e.Record.Items()
		for i := 0; i < items; i++ {
			if !e.Record.Next() {
				return 0, e.Record.Error()
			}
			key := e.Record.Str()

			if !e.Record.Next() {
				return 0, e.Record.Error()
			}

			b = append(b, ' ')
			b = append(b, fast.StringToBytes(key)...)
			b = append(b, '=')
			b, _ = e.Record.AppendText(b)
		}

		b = append(b, '\n')

		if _, err = a.w.Write(b); err != nil {
			return 0, err
		}

		// keep buffer for reuse
		a.buf = b[:0]
	}
}
