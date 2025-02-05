package fluentlog

import (
	"runtime"
	"strconv"
	"sync"
	"unsafe"

	"github.com/webmafia/fluentlog/internal/msgpack"
)

var stackTracePool sync.Pool

func StackTrace(skip ...int) KeyValueAppender {
	s := 2

	if len(skip) > 0 && skip[0] > 0 {
		s += skip[0]
	}

	trace, ok := stackTracePool.Get().(*stackTrace)

	if !ok {
		trace = new(stackTrace)
		trace.frames.frames = trace.frames.frameStore[:0]
	}

	n := runtime.Callers(s, trace.callers[:])
	trace.frames.callers = trace.callers[:n]

	return trace
}

// Must exactly match runtime.Frames.
type stackTrace struct {
	frames
	callers [16]uintptr
}

type frames struct {
	// callers is a slice of PCs that have not yet been expanded to frames.
	callers []uintptr

	// nextPC is a next PC to expand ahead of processing callers.
	_ uintptr

	// frames is a slice of Frames that have yet to be returned.
	frames     []runtime.Frame
	frameStore [2]runtime.Frame
}

// AppendKeyValue implements KeyValueAppender.
func (trace *stackTrace) AppendKeyValue(dst []byte, key string) ([]byte, uint8) {
	frames := (*runtime.Frames)(unsafe.Pointer(trace))
	var n uint8

	var (
		frame runtime.Frame
		more  = true
	)

	if key == "" {
		key = "stackTrace"
	}

	for more {
		frame, more = frames.Next()

		dst = msgpack.AppendString(dst, key)
		dst = msgpack.AppendStringMax255(dst, func(dst []byte) []byte {
			dst = append(dst, frame.File...)
			dst = append(dst, ':')
			dst = strconv.AppendInt(dst, int64(frame.Line), 10)
			return dst
		})

		n++
	}

	// Reset and put back to pool
	trace.frames.frames = trace.frames.frameStore[:0]
	stackTracePool.Put(trace)

	return dst, n
}
