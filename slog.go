package fluentlog

import (
	"context"
	"log/slog"
	"time"

	"github.com/webmafia/fast"
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/hexid"
)

func (l *Logger) SlogHandler() slog.Handler {
	return slogHandler{l: l}
}

var _ slog.Handler = slogHandler{}

// A Handler handles log records produced by a Logger.
//
// A typical handler may print log records to standard error,
// or write them to a file or database, or perhaps augment them
// with additional attributes and pass them on to another handler.
//
// Any of the Handler's methods may be called concurrently with itself
// or with other methods. It is the responsibility of the Handler to
// manage this concurrency.
//
// Users of the slog package should not invoke Handler methods directly.
// They should use the methods of [Logger] instead.
type slogHandler struct {
	l *Logger
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
// It is called early, before any arguments are processed,
// to save effort if the log event should be discarded.
// If called from a Logger method, the first argument is the context
// passed to that method, or context.Background() if nil was passed
// or the method does not take a context.
// The context is passed so Enabled can use its values
// to make a decision.
func (s slogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// The Context argument is as for Enabled.
// It is present solely to provide Handlers access to the context's values.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
//
// Handle methods that produce output should observe the following rules:
//   - If r.Time is the zero time, ignore the time.
//   - If r.PC is zero, ignore it.
//   - Attr's values should be resolved.
//   - If an Attr's key and value are both the zero value, ignore the Attr.
//     This can be tested with attr.Equal(Attr{}).
//   - If a group's key is empty, inline the group's Attrs.
//   - If a group has no Attrs (even if it has a non-empty key),
//     ignore it.
func (s slogHandler) Handle(ctx context.Context, rec slog.Record) error {
	var sev Severity = INFO

	switch rec.Level {
	case slog.LevelDebug:
		sev = DEBUG
	case slog.LevelInfo:
		sev = INFO
	case slog.LevelWarn:
		sev = WARN
	case slog.LevelError:
		sev = ERR
	}

	s.log(sev, rec.Message, fast.Noescape(&rec), 4, s.l.fieldData, s.l.fieldCount)

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (s slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	log := s.l.inst.Logger()
	log.fieldData = s.l.inst.bufPool.Get()

	if s.l.fieldData != nil {
		log.fieldData.B = append(log.fieldData.B, s.l.fieldData.B...)
		log.fieldCount = s.l.fieldCount
	}

	for i := range attrs {
		appendSlogAttr(log.fieldData, fast.Noescape(&attrs[i]))
	}

	log.fieldCount += uint8(len(attrs))

	return slogHandler{l: log}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
//
// How this qualification happens is up to the Handler, so long as
// this Handler's attribute keys differ from those of another Handler
// with a different sequence of group names.
//
// A Handler should treat WithGroup as starting a Group of Attrs that ends
// at the end of the log event. That is,
//
//	logger.WithGroup("s").LogAttrs(ctx, level, msg, slog.Int("a", 1), slog.Int("b", 2))
//
// should behave like
//
//	logger.LogAttrs(ctx, level, msg, slog.Group("s", slog.Int("a", 1), slog.Int("b", 2)))
//
// If the name is empty, WithGroup returns the receiver.
func (s slogHandler) WithGroup(name string) slog.Handler {

	// TODO: Respect the group name. We do NOT want to add it directly into slogHandler,
	// as that would increase its size from 8 bytes (which fits in an interface) to 24 bytes
	// (that forces Go to wrap the slogHandler with a pointer to bring it down to 8 bytes).
	return slogHandler{l: s.l.inst.Logger()}
}

func (s slogHandler) log(sev Severity, msg string, rec *slog.Record, skipStackTrace int, extraData *buffer.Buffer, extraCount uint8) (id hexid.ID) {
	if s.l.inst.closed() {
		return
	}

	ts := rec.Time

	if ts.IsZero() {
		ts = time.Now()
	}

	b := s.l.inst.bufPool.Get()
	id = hexid.IDFromTime(ts)

	b.B = msgpack.AppendArrayHeader(b.B, 3)
	b.B = msgpack.AppendString(b.B, s.l.inst.opt.Tag)
	b.B = msgpack.AppendTimestamp(b.B, ts, msgpack.TsFluentd)
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
	b.B = msgpack.AppendString(b.B, msg)

	if extraCount > 0 {
		b.B = append(b.B, extraData.B...)
		b.B[x] += extraCount
	}

	b.B[x] += appendSlogAttrs(b, fast.Noescape(rec))

	if sev <= s.l.inst.opt.StackTraceThreshold {
		b.B = appendStackTrace(b.B, skipStackTrace)
		b.B[x]++
	}

	s.l.inst.queueMessage(b)
	return
}

func appendSlogAttrs(b *buffer.Buffer, rec *slog.Record) (n uint8) {
	rec.Attrs(func(a slog.Attr) bool {
		appendSlogAttr(b, fast.Noescape(&a))

		return true
	})

	return uint8(rec.NumAttrs())
}

func appendSlogAttr(b *buffer.Buffer, a *slog.Attr) {
	b.B = msgpack.AppendString(b.B, a.Key)

	switch a.Value.Kind() {
	case slog.KindInt64:
		b.B = msgpack.AppendInt(b.B, a.Value.Int64())
	case slog.KindUint64:
		b.B = msgpack.AppendUint(b.B, a.Value.Uint64())
	case slog.KindFloat64:
		b.B = msgpack.AppendFloat(b.B, a.Value.Float64())
	case slog.KindBool:
		b.B = msgpack.AppendBool(b.B, a.Value.Bool())
	case slog.KindTime:
		b.B = msgpack.AppendTimestamp(b.B, a.Value.Time())
	case slog.KindString:
		b.B = msgpack.AppendString(b.B, a.Value.String())
	default:
		b.B = msgpack.AppendAny(b.B, a.Value.Any())
	}
}
