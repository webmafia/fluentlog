package fluentlog

import (
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/identifier"
)

type Logger struct {
	cli    io.Writer
	pool   buffer.Pool
	ch     chan *buffer.Buffer
	wg     sync.WaitGroup
	closed atomic.Bool
}

func NewLogger(cli io.Writer, bufferSize ...int) *Logger {
	bufSize := 16

	if len(bufferSize) > 0 && bufferSize[0] >= 0 {
		bufSize = bufferSize[0]
	}

	l := &Logger{
		cli: cli,
		ch:  make(chan *buffer.Buffer, bufSize),
	}

	l.wg.Add(1)
	go l.worker()

	return l
}

func (l *Logger) Close() {
	if l.closed.Swap(true) {
		return
	}

	close(l.ch)
	l.wg.Wait()

	if closer, ok := l.cli.(io.WriteCloser); ok {
		closer.Close()
	}
}

func (l *Logger) Log(msg string, meta ...any) (id identifier.ID) {
	if l.closed.Load() {
		return
	}

	b := l.pool.Get()
	id = identifier.Generate()

	b.B = msgpack.AppendArrayHeader(b.B, 3)
	b.B = msgpack.AppendString(b.B, "foo.bar")
	b.B = msgpack.AppendTimestamp(b.B, id.Time(), msgpack.TsFluentd)

	n := len(meta)
	n -= n % 2

	b.B = msgpack.AppendMapHeader(b.B, (n/2)+2)

	b.B = msgpack.AppendString(b.B, "@id")
	b.B = msgpack.AppendTextAppender(b.B, id)

	b.B = msgpack.AppendString(b.B, "message")
	b.B = msgpack.AppendString(b.B, msg)

	for i := 0; i < n; i++ {
		b.B = msgpack.AppendAny(b.B, meta[i])
	}

	// Try to write 10 times per capacity of the channel
	// if !tryWrite(l.ch, b, cap(l.ch)*100) {
	// 	l.pool.Put(b)
	// 	id = 0
	// }

	l.ch <- b

	return
}

func (l *Logger) worker() {
	defer l.wg.Done()

	for b := range l.ch {
		err := l.write(b.B)

		if err != nil {
			log.Println(err)
		}

		l.pool.Put(b)
	}
}

func (log *Logger) write(b []byte) (err error) {
	_, err = log.cli.Write(fast.NoescapeBytes(b))
	return
}
