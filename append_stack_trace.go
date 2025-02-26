package fluentlog

import (
	"runtime"
	"strconv"
	"unsafe"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

// An exact copy of runtime.Frames
type frames struct {
	// callers is a slice of PCs that have not yet been expanded to frames.
	callers []uintptr

	// nextPC is a next PC to expand ahead of processing callers.
	_ uintptr

	// frames is a slice of Frames that have yet to be returned.
	frames     []runtime.Frame
	frameStore [2]runtime.Frame
}

func appendStackTrace(dst []byte, skip int) []byte {
	var callers [15]uintptr
	n := runtime.Callers(skip, callers[:])
	f := frames{callers: callers[:n]}
	f.frames = f.frameStore[:0]
	frames := (*runtime.Frames)(fast.Noescape(unsafe.Pointer(&f)))

	var (
		frame runtime.Frame
		more  = true
	)

	var i uint8

	dst = msgpack.AppendString(dst, "stackTrace")
	dst = msgpack.AppendArrayHeader(dst, 0)
	x := len(dst) - 1

	for more {
		frame, more = frames.Next()
		dst = msgpack.AppendStringMax255(dst, func(dst []byte) []byte {
			dst = append(dst, frame.File...)
			dst = append(dst, ':')
			dst = strconv.AppendInt(dst, int64(frame.Line), 10)
			return dst
		})

		i++
	}

	dst[x] = i

	return dst
}
