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
	opt    Options
	pool   buffer.Pool
	ch     chan *buffer.Buffer
	wg     sync.WaitGroup
	closed atomic.Bool
}

type Options struct {
	Tag        string
	BufferSize int
}

func (opt *Options) setDefaults() {
	if opt.Tag == "" {
		opt.Tag = "fluentlog"
	}

	if opt.BufferSize <= 0 {
		opt.BufferSize = 16
	}
}

func NewLogger(cli io.Writer, options ...Options) *Logger {
	var opt Options

	if len(options) > 0 {
		opt = options[0]
	}

	opt.setDefaults()

	l := &Logger{
		cli: cli,
		opt: opt,
		ch:  make(chan *buffer.Buffer, opt.BufferSize),
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

func (l *Logger) Debug(msg string, args ...any) identifier.ID { return l.log(DEBUG, msg, args) }
func (l *Logger) Info(msg string, args ...any) identifier.ID  { return l.log(INFO, msg, args) }
func (l *Logger) Warn(msg string, args ...any) identifier.ID  { return l.log(WARNING, msg, args) }
func (l *Logger) Error(msg string, args ...any) identifier.ID { return l.log(ERR, msg, args) }

func (l *Logger) log(sev Severity, msg string, args []any) (id identifier.ID) {
	if l.closed.Load() {
		return
	}

	b := l.pool.Get()
	id = identifier.Generate()

	b.B = msgpack.AppendArrayHeader(b.B, 3)
	b.B = msgpack.AppendString(b.B, l.opt.Tag)
	b.B = msgpack.AppendTimestamp(b.B, id.Time(), msgpack.TsFluentd)
	b.B = append(b.B, 0xde, 0, 0) // map 16

	x := len(b.B) - 1

	b.B[x]++
	b.B = msgpack.AppendString(b.B, "@id")
	b.B = msgpack.AppendInt(b.B, id.Int64())

	b.B[x]++
	b.B = msgpack.AppendString(b.B, "@pri")
	b.B = msgpack.AppendUint(b.B, uint64(sev))

	b.B[x]++
	b.B = msgpack.AppendString(b.B, "message")
	b.B = msgpack.AppendString(b.B, msg)

	var key string
	var n uint8

	for i := range args {
		if key == "" {
			if k, ok := args[i].(string); ok {
				key = k
				continue
			}
		}

		b.B, n = appendKeyValue(b.B, key, args[i])
		key = ""
		b.B[x] += n
	}

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
