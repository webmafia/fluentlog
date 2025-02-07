package fluentlog

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/fallback"
	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/identifier"
)

type Instance struct {
	cli        io.Writer
	opt        Options
	bufPool    buffer.Pool         // Pool of buffers
	logPool    sync.Pool           // Pool of loggers
	queue      chan *buffer.Buffer // Main queue (buffered)
	fbQueue    chan *buffer.Buffer // Fallback queue (unbuffered)
	fbNonEmpty chan struct{}       // Whether fbQueue is non-empty (exactly 1 in buffer size)
	close      chan struct{}       // Close channel
	wg         sync.WaitGroup
}

type Options struct {
	Tag           string
	BufferSize    int
	WriteBehavior WriteBehavior
	Fallback      *fallback.DirBuffer
}

func (opt *Options) setDefaults() {
	if opt.Tag == "" {
		opt.Tag = "fluentlog"
	}

	if opt.BufferSize <= 0 {
		opt.BufferSize = 16
	}
}

func NewInstance(cli io.Writer, options ...Options) (*Instance, error) {
	var opt Options

	if len(options) > 0 {
		opt = options[0]
	}

	opt.setDefaults()

	inst := &Instance{
		cli:   cli,
		opt:   opt,
		queue: make(chan *buffer.Buffer, opt.BufferSize),
	}

	if inst.opt.WriteBehavior == Fallback {
		if inst.opt.Fallback == nil {
			return nil, errors.New("WriteBehavior set to 'Fallback', but not Fallblack provided")
		}

		if _, ok := inst.cli.(BatchWriter); !ok {
			return nil, errors.New("WriteBehavior set to 'Fallback', but client doesn't implement BatchWriter")
		}

		inst.fbQueue = make(chan *buffer.Buffer)
		inst.fbNonEmpty = make(chan struct{}, 1)

		inst.wg.Add(1)
		go inst.fallbackWorker()
	}

	inst.wg.Add(1)
	go inst.worker()

	return inst, nil
}

func (inst *Instance) Logger() *Logger {
	l, ok := inst.logPool.Get().(*Logger)

	if !ok {
		l = &Logger{
			inst: inst,
		}
	}

	return l
}

func (inst *Instance) Release(l *Logger) {
	inst.bufPool.Put(l.fieldData)
	l.fieldData = nil
	l.fieldCount = 0
	inst.logPool.Put(l)
}

func (inst *Instance) Close() {
	close(inst.close)
	inst.wg.Wait()

	if closer, ok := inst.cli.(io.WriteCloser); ok {
		closer.Close()
	}
}

func (inst *Instance) closed() bool {
	select {
	case <-inst.close:
		return true
	default:
		return false
	}
}

func (inst *Instance) log(sev Severity, msg string, args []any, extraData *buffer.Buffer, extraCount uint8) (id identifier.ID) {
	if inst.closed() {
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
		b.B = append(b.B, extraData.B...)
		b.B[x] += extraCount
	}

	b.B[x] += appendArgs(b, args)

	inst.queueMessage(b)
	return
}

func (inst *Instance) queueMessage(b *buffer.Buffer) {
	if inst.opt.WriteBehavior == Block {
		inst.queue <- b
		return
	}

	select {

	// Try to put message in queue
	case inst.queue <- b:

	// If the queue is full
	default:
		if inst.opt.WriteBehavior == Fallback {
			inst.fbQueue <- b
		} else {
			inst.bufPool.Put(b)
		}
	}
}

func (inst *Instance) worker() {
	defer inst.wg.Done()

	for {
		select {

		case b := <-inst.queue:
			if err := inst.write(b.B); err != nil {
				log.Println(err)
			}

			inst.bufPool.Put(b)

		case <-inst.close:
			for {
				select {

				case b := <-inst.queue:
					if err := inst.write(b.B); err != nil {
						log.Println(err)
					}

					inst.bufPool.Put(b)

				default:
					return

				}
			}
		}
	}
}

func (inst *Instance) fallbackWorker() {
	defer inst.wg.Done()

	for b := range inst.fbQueue {

		inst.bufPool.Put(b)

		// Tell that there are messages in the fallback buffer
		select {
		case inst.fbNonEmpty <- struct{}{}:
		default:
		}
	}
}

func (inst *Instance) write(b []byte) (err error) {
	_, err = inst.cli.Write(fast.NoescapeBytes(b))
	return
}
