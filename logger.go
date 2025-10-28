package fluentlog

import (
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/hexid"
)

// A logger with optional meta data, which every log entry will inherit.
type Logger struct {
	inst       *Instance
	fieldData  *buffer.Buffer
	fieldCount uint8
}

// Acquires a new, empty logger.
func NewLogger(inst *Instance) *Logger {
	return inst.Logger()
}

// Debug or trace information.
func (l *Logger) Debug(msg string, args ...any) hexid.ID {
	return l.inst.log(DEBUG, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// Routine information, such as ongoing status or performance.
func (l *Logger) Info(msg string, args ...any) hexid.ID {
	return l.inst.log(INFO, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// Normal but significant events, such as start up, shut down, or a configuration change.
func (l *Logger) Notice(msg string, args ...any) hexid.ID {
	return l.inst.log(NOTICE, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// Warning events might cause problems.
func (l *Logger) Warn(msg string, args ...any) hexid.ID {
	return l.inst.log(WARN, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// Error events are likely to cause problems.
func (l *Logger) Error(msg string, args ...any) hexid.ID {
	return l.inst.log(ERR, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// Critical events cause more severe problems or outages.
func (l *Logger) Crit(msg string, args ...any) hexid.ID {
	return l.inst.log(CRIT, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// A person must take an action immediately.
func (l *Logger) Alert(msg string, args ...any) hexid.ID {
	return l.inst.log(ALERT, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// One or more systems are unusable.
func (l *Logger) Emerg(msg string, args ...any) hexid.ID {
	return l.inst.log(EMERG, msg, args, false, 4, l.fieldData, l.fieldCount)
}

// Debug or trace information. Formatted with printf syntax.
func (l *Logger) Debugf(format string, args ...any) hexid.ID {
	return l.inst.log(DEBUG, format, args, true, 4, l.fieldData, l.fieldCount)
}

// Routine information, such as ongoing status or performance. Formatted with printf syntax.
func (l *Logger) Infof(format string, args ...any) hexid.ID {
	return l.inst.log(INFO, format, args, true, 4, l.fieldData, l.fieldCount)
}

// Normal but significant events, such as start up, shut down, or a configuration change. Formatted with printf syntax.
func (l *Logger) Noticef(msg string, args ...any) hexid.ID {
	return l.inst.log(NOTICE, msg, args, true, 4, l.fieldData, l.fieldCount)
}

// Warning events might cause problems. Formatted with printf syntax.
func (l *Logger) Warnf(format string, args ...any) hexid.ID {
	return l.inst.log(WARN, format, args, true, 4, l.fieldData, l.fieldCount)
}

// Error events are likely to cause problems. Formatted with printf syntax.
func (l *Logger) Errorf(format string, args ...any) hexid.ID {
	return l.inst.log(ERR, format, args, true, 4, l.fieldData, l.fieldCount)
}

// Critical events cause more severe problems or outages. Formatted with printf syntax.
func (l *Logger) Critf(msg string, args ...any) hexid.ID {
	return l.inst.log(CRIT, msg, args, true, 4, l.fieldData, l.fieldCount)
}

// A person must take an action immediately. Formatted with printf syntax.
func (l *Logger) Alertf(msg string, args ...any) hexid.ID {
	return l.inst.log(ALERT, msg, args, true, 4, l.fieldData, l.fieldCount)
}

// One or more systems are unusable. Formatted with printf syntax.
func (l *Logger) Emergf(msg string, args ...any) hexid.ID {
	return l.inst.log(EMERG, msg, args, true, 4, l.fieldData, l.fieldCount)
}

// Logs metric values. Example usage:
//
//	log.Warn("hello world",
//	    "myKey", 123,
//	    "otherKey", 456,
//	)
func (l *Logger) Metrics(args ...any) {
	l.inst.metrics(args)
}

// Acquires a new logger with meta data, that inherits any meta data from
// the current logger. The new logger is returned, and should be released once
// finished. Example usage:
//
//	sub := log.With(
//	    "myKey", "myValue",
//	)
//	defer sub.Release()
func (l *Logger) With(args ...any) *Logger {
	log := l.inst.Logger()

	if len(args) > 0 {
		log.fieldData = l.inst.bufPool.Get()

		if l.fieldData != nil {
			log.fieldData.B = append(log.fieldData.B, l.fieldData.B...)
			log.fieldCount = l.fieldCount
		}

		log.fieldCount += appendArgs(log.fieldData, args)
	}

	return log
}

// Releases the logger for reuse.
func (l *Logger) Release() {
	l.inst.Release(l)
}

// Recovers from a panic and logs it as a critical error. Usage:
//
//	go func() {
//	    defer log.Recover()
//
//	    panic("aaaaaahh")
//	}()
func (l *Logger) Recover() {
	err := recover()
	l.inst.log(CRIT, "panic: %v", []any{err}, true, 5, l.fieldData, l.fieldCount)
}
