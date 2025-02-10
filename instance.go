package fluentlog

import (
	"errors"
	"fmt"
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
	Tag                 string
	BufferSize          int
	WriteBehavior       WriteBehavior
	Fallback            *fallback.DirBuffer
	StackTraceThreshold Severity
}

func (opt *Options) setDefaults() {
	if opt.Tag == "" {
		opt.Tag = "fluentlog"
	}

	if opt.BufferSize <= 0 {
		opt.BufferSize = 16
	}
}

// Create a logger instance used for acquiring loggers.
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
		close: make(chan struct{}),
	}

	if inst.opt.WriteBehavior == Fallback {
		if inst.opt.Fallback == nil {
			return nil, errors.New("WriteBehavior set to 'Fallback', but no Fallblack provided")
		}

		if _, ok := inst.cli.(BatchWriter); !ok {
			return nil, errors.New("WriteBehavior set to 'Fallback', but client doesn't implement BatchWriter")
		}

		inst.fbQueue = make(chan *buffer.Buffer)
		inst.fbNonEmpty = make(chan struct{}, 1)

		inst.wg.Add(1)
		go inst.fallbackWorker()

		ok, err := inst.opt.Fallback.AnythingToRead()

		if err != nil {
			return nil, err
		}

		if ok {
			inst.fbNonEmpty <- struct{}{}
		}
	}

	inst.wg.Add(1)
	go inst.worker()

	return inst, nil
}

// Acquires a new, empty logger.
func (inst *Instance) Logger() *Logger {
	l, ok := inst.logPool.Get().(*Logger)

	if !ok {
		l = &Logger{
			inst: inst,
		}
	}

	return l
}

// Releases a logger for reuse.
func (inst *Instance) Release(l *Logger) {
	inst.bufPool.Put(l.fieldData)
	l.fieldData = nil
	l.fieldCount = 0
	inst.logPool.Put(l)
}

// Closes the instance. Any new log entries will be ignored, while entries already written
// will be processed. Blocks until fully drained.
func (inst *Instance) Close() (err error) {
	close(inst.close)
	inst.wg.Wait()

	if inst.opt.Fallback != nil {
		if err = inst.opt.Fallback.Close(); err != nil {
			return
		}
	}

	if closer, ok := inst.cli.(io.WriteCloser); ok {
		err = closer.Close()
	}

	return
}

func (inst *Instance) closed() bool {
	select {
	case <-inst.close:
		return true
	default:
		return false
	}
}

func (inst *Instance) log(sev Severity, msg string, args []any, sprintf bool, skipStackTrace int, extraData *buffer.Buffer, extraCount uint8) (id identifier.ID) {
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
	if sprintf {
		b.B = msgpack.AppendStringDynamic(b.B, func(dst []byte) []byte {
			return fmt.Appendf(dst, msg, args...)
		})
	} else {
		b.B = msgpack.AppendString(b.B, msg)
	}

	if extraCount > 0 {
		b.B = append(b.B, extraData.B...)
		b.B[x] += extraCount
	}

	if !sprintf {
		b.B[x] += appendArgs(b, args)
	}

	if sev <= inst.opt.StackTraceThreshold {
		var n uint8
		b.B, n = appendStackTrace(b.B, skipStackTrace)
		b.B[x] += n
	}

	inst.queueMessage(b)
	return
}

func (inst *Instance) queueMessage(b *buffer.Buffer) {
	select {

	// Try to put message in queue
	case inst.queue <- b:

	// If the queue is full
	default:
		switch inst.opt.WriteBehavior {
		case Block:
			inst.queue <- b
		case Fallback:
			inst.fbQueue <- b
		default:
			inst.bufPool.Put(b)
		}
	}
}

// The worker prioritizes any pending log messages in queue. If it's empty,
// it's checks for any fallback buffer and handles it.
func (inst *Instance) worker() {
	defer inst.wg.Done()

	for {
		// First, try a non-blocking receive from the main queue.
		select {
		case b := <-inst.queue:
			inst.sendToCli(b)
			continue
		default:
			// Nothing immediately available on the main queue.
		}

		// Now block waiting for a log message, a fallback signal, or a shutdown.
		select {
		case b := <-inst.queue:
			inst.sendToCli(b)

		case <-inst.fbNonEmpty:
			err := inst.opt.Fallback.Reader(func(size int, r io.Reader) error {
				return inst.cli.(BatchWriter).WriteBatch(inst.opt.Tag, size, r)
			})

			if err != nil {
				log.Println(err)
			}

		case <-inst.close:
			// Shutdown has been signaled.
			// Drain any remaining messages on the main queue.
			for {
				select {
				case b := <-inst.queue:
					inst.sendToCli(b)
				default:
					// No more messages; exit the worker.
					return
				}
			}
		}
	}
}

func (inst *Instance) sendToCli(b *buffer.Buffer) {
	if _, err := inst.cli.Write(fast.NoescapeBytes(b.B)); err != nil {
		log.Println("error while writing to cli:", err)

		if inst.opt.WriteBehavior == Fallback {
			inst.fbQueue <- b
			return
		}
	}

	inst.bufPool.Put(b)
}

func (inst *Instance) fallbackWorker() {
	defer inst.wg.Done()

	for {
		select {
		case b := <-inst.fbQueue:
			inst.sendToFallbackCli(b)
		case <-inst.close:
			// Shutdown has been signaled.
			// Drain any remaining messages on the fallback queue.
			for {
				select {
				case b := <-inst.queue:
					inst.sendToFallbackCli(b)
				default:
					// No more messages; exit the worker.
					return
				}
			}
		}

	}
}

func (inst *Instance) sendToFallbackCli(b *buffer.Buffer) {

	// When writing to fallback, each entry should only consist of an array of 2 items (timestamp + record).
	// For this reason, we must strip away the original array header + the tag string, then write an array
	// header of 2 items + the rest of the entry.
	strip := 1 + strSize(len(inst.opt.Tag))
	inst.opt.Fallback.Write([]byte{0x90 | 2})
	_, err := inst.opt.Fallback.Write(fast.NoescapeBytes(b.B[strip:]))

	inst.bufPool.Put(b)

	// Tell that there are messages in the fallback buffer
	if err != nil {
		log.Println(err)
	} else {
		select {
		case inst.fbNonEmpty <- struct{}{}:
		default:
		}
	}
}

func strSize(l int) int {
	switch {
	case l <= 31:
		l++
	case l <= 0xFF:
		l += 2
	case l <= 0xFFFF:
		l += 3
	default:
		l += 5
	}

	return l
}
