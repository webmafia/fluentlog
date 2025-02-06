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

type Instance struct {
	cli     io.Writer
	opt     Options
	bufPool buffer.Pool
	logPool sync.Pool
	ch      chan *buffer.Buffer
	wg      sync.WaitGroup
	closed  atomic.Bool
}

func NewInstance(cli io.Writer, options ...Options) *Instance {
	var opt Options

	if len(options) > 0 {
		opt = options[0]
	}

	opt.setDefaults()

	inst := &Instance{
		cli: cli,
		opt: opt,
		ch:  make(chan *buffer.Buffer, opt.BufferSize),
	}

	inst.wg.Add(1)
	go inst.worker()

	return inst
}

func (inst *Instance) Logger() *Logger {
	l, ok := inst.logPool.Get().(*Logger)

	if !ok {
		l = &Logger{
			inst: inst,
		}
	}

	l.fieldData = inst.bufPool.Get()

	return l
}

func (inst *Instance) Release(l *Logger) {
	inst.bufPool.Put(l.fieldData)
	l.fieldData = nil
	l.fieldCount = 0
	inst.logPool.Put(l)
}

func (inst *Instance) Close() {
	if inst.closed.Swap(true) {
		return
	}

	close(inst.ch)
	inst.wg.Wait()

	if closer, ok := inst.cli.(io.WriteCloser); ok {
		closer.Close()
	}
}

func (inst *Instance) log(sev Severity, msg string, args []any, extraData []byte, extraCount uint8) (id identifier.ID) {
	if inst.closed.Load() {
		return
	}

	b := inst.bufPool.Get()
	id = identifier.Generate()

	b.B = msgpack.AppendArrayHeader(b.B, 3)
	b.B = msgpack.AppendString(b.B, inst.opt.Tag)
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

	if extraCount > 0 {
		b.B = append(b.B, extraData...)
		b.B[x] += extraCount
	}

	b.B[x] += appendArgs(b, args)

	inst.ch <- b
	return
}

func (inst *Instance) worker() {
	defer inst.wg.Done()

	for b := range inst.ch {
		err := inst.write(b.B)

		if err != nil {
			log.Println(err)
		}

		inst.logPool.Put(b)
	}
}

func (inst *Instance) write(b []byte) (err error) {
	_, err = inst.cli.Write(fast.NoescapeBytes(b))
	return
}
