package fluentlog

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/fallback"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/hexid"
)

type Instance struct {
	cli     io.Writer
	opt     Options
	bufPool buffer.Pool         // Pool of buffers
	logPool sync.Pool           // Pool of loggers
	queue   chan *buffer.Buffer // Main queue (buffered)
	close   chan struct{}       // Close channel
	done    chan struct{}       // Done channel
	wg      sync.WaitGroup
	fb      bool
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

type Reconnector interface {
	Reconnect() error
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
		done:  make(chan struct{}),
	}

	if inst.opt.WriteBehavior == Fallback {
		if inst.opt.Fallback == nil {
			return nil, errors.New("WriteBehavior set to 'Fallback', but no Fallblack provided")
		}

		if _, ok := inst.cli.(BatchWriter); !ok {
			return nil, errors.New("WriteBehavior set to 'Fallback', but client doesn't implement BatchWriter")
		}

		// inst.wg.Add(1)
		// go inst.fallbackWorker()
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

func (inst *Instance) log(sev Severity, msg string, args []any, sprintf bool, skipStackTrace int, extraData *buffer.Buffer, extraCount uint8) (id hexid.ID) {
	var fmtArgs int

	if sprintf {
		fmtArgs = countFmtArgs(msg)
	}

	if inst.closed() {
		return
	}

	b := inst.bufPool.Get()
	id = hexid.Generate()

	b.B = msgpack.AppendArrayHeader(b.B, 3)
	b.B = msgpack.AppendString(b.B, inst.opt.Tag)
	b.B = msgpack.AppendTimestamp(b.B, id.Time(), msgpack.TsFluentd)
	b.B = append(b.B, 0xde, 0, 0) // map 16

	x := len(b.B) - 1

	b.B[x]++
	b.B = msgpack.AppendString(b.B, "@id")
	b.B = msgpack.AppendUint(b.B, id.Uint64())

	b.B[x]++
	b.B = msgpack.AppendString(b.B, "pri")
	b.B = msgpack.AppendUint(b.B, uint64(sev))

	b.B[x]++
	b.B = msgpack.AppendString(b.B, "message")
	if fmtArgs > 0 {
		b.B = msgpack.AppendStringDynamic(b.B, func(dst []byte) []byte {
			return fmt.Appendf(dst, msg, args[:fmtArgs]...)
		})
	} else {
		b.B = msgpack.AppendString(b.B, msg)
	}

	if extraCount > 0 {
		b.B = append(b.B, extraData.B...)
		b.B[x] += extraCount
	}

	if len(args) > fmtArgs {
		b.B[x] += appendArgs(b, args[fmtArgs:])
	}

	if sev <= inst.opt.StackTraceThreshold {
		b.B = appendStackTrace(b.B, skipStackTrace)
		b.B[x]++
	}

	inst.queueMessage(b)
	return
}

func (inst *Instance) metrics(args []any) {
	if inst.closed() {
		return
	}

	b := inst.bufPool.Get()

	b.B = msgpack.AppendArrayHeader(b.B, 3)
	b.B = msgpack.AppendString(b.B, inst.opt.Tag)
	b.B = msgpack.AppendTimestamp(b.B, time.Now(), msgpack.TsFluentd)
	b.B = append(b.B, 0xde, 0, 0) // map 16

	x := len(b.B) - 1
	b.B[x] += appendArgs(b, args)

	inst.queueMessage(b)
}

func (inst *Instance) queueMessage(b *buffer.Buffer) {
	select {

	// Try to put message in queue
	case inst.queue <- b:

	// If the queue is full
	default:
		switch inst.opt.WriteBehavior {
		case Block, Fallback:
			inst.queue <- b
		default:
			inst.bufPool.Put(b)
		}
	}
}

// The worker prioritizes any pending log messages in queue. If it's empty,
// it checks for any fallback buffer and handles it.
func (inst *Instance) worker() {
	defer func() {
		close(inst.done)
		inst.wg.Done()
	}()

	if err := inst.maybeSetFb(); err != nil {
		log.Println("failed to set fb")
	}

	if err := inst.flushFallbackToCli(); err != nil {
		log.Println("failed to flush fallback to cli:", err)
	}

	fallbackTicker := time.NewTicker(10 * time.Second)

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

		case <-fallbackTicker.C:
			if err := inst.flushFallbackToCli(); err != nil {
				log.Println("failed to flush fallback to cli:", err)
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
	if inst.fb {
		inst.sendToFallbackCli(b)
		return
	}

	if _, err := inst.cli.Write(fast.Noescape(b.B)); err != nil {
		log.Println("error while writing to cli:", err)

		if inst.opt.WriteBehavior == Fallback {
			inst.fb = true
			inst.sendToFallbackCli(b)
			return
		}
	}

	inst.bufPool.Put(b)
}

func (inst *Instance) sendToFallbackCli(b *buffer.Buffer) {

	// When writing to fallback, each entry should only consist of an array of 2 items (timestamp + record).
	// For this reason, we must strip away the original array header + the tag string, then write an array
	// header of 2 items + the rest of the entry.
	strip := 1 + strSize(len(inst.opt.Tag))
	inst.opt.Fallback.Write([]byte{0x90 | 2})
	_, err := inst.opt.Fallback.Write(fast.Noescape(b.B[strip:]))

	inst.bufPool.Put(b)

	// Tell that there are messages in the fallback buffer
	if err != nil {
		log.Println(err)
	}
}

func (inst *Instance) flushFallbackToCli() (err error) {
	if !inst.fb {
		return
	}

	err = inst.opt.Fallback.Reader(func(size int, r io.Reader) error {
		return inst.cli.(BatchWriter).WriteBatch(inst.opt.Tag, size, r)
	})

	if err != nil {
		if cli, ok := inst.cli.(Reconnector); ok {
			if err = cli.Reconnect(); err != nil {
				return
			}

			if err = inst.opt.Fallback.Reader(func(size int, r io.Reader) error {
				return inst.cli.(BatchWriter).WriteBatch(inst.opt.Tag, size, r)
			}); err != nil {
				return
			}
		}
	}

	inst.fb = false

	return
}

func (inst *Instance) maybeSetFb() (err error) {
	if inst.opt.WriteBehavior == Fallback {
		inst.fb, err = inst.opt.Fallback.HasData()
	}

	return
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
