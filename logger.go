package fluentlog

import (
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/identifier"
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

// Logs a new entry of "Debug" severity, with optional meta data. Example usage:
//
//	log.Debug("hello world",
//	    "myKey", "myValue",
//	    "otherKey", 123,
//	)
func (l *Logger) Debug(msg string, args ...any) identifier.ID {
	return l.inst.log(DEBUG, msg, args, false, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Info" severity, with optional meta data. Example usage:
//
//	log.Info("hello world",
//	    "myKey", "myValue",
//	    "otherKey", 123,
//	)
func (l *Logger) Info(msg string, args ...any) identifier.ID {
	return l.inst.log(INFO, msg, args, false, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Warning" severity, with optional meta data. Example usage:
//
//	log.Warn("hello world",
//	    "myKey", "myValue",
//	    "otherKey", 123,
//	)
func (l *Logger) Warn(msg string, args ...any) identifier.ID {
	return l.inst.log(WARN, msg, args, false, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Error" severity, with optional meta data. Example usage:
//
//	log.Error("hello world",
//	    "myKey", "myValue",
//	    "otherKey", 123,
//	)
func (l *Logger) Error(msg string, args ...any) identifier.ID {
	return l.inst.log(ERR, msg, args, false, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Debug" severity, with a formatted (printf) message. Example usage:
//
//	log.Debugf("the number is %d", 123)
func (l *Logger) Debugf(format string, args ...any) identifier.ID {
	return l.inst.log(DEBUG, format, args, true, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Info" severity, with a formatted (printf) message. Example usage:
//
//	log.Infof("the number is %d", 123)
func (l *Logger) Infof(format string, args ...any) identifier.ID {
	return l.inst.log(INFO, format, args, true, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Warning" severity, with a formatted (printf) message. Example usage:
//
//	log.Warnf("the number is %d", 123)
func (l *Logger) Warnf(format string, args ...any) identifier.ID {
	return l.inst.log(WARN, format, args, true, 3, l.fieldData, l.fieldCount)
}

// Logs a new entry of "Error" severity, with a formatted (printf) message. Example usage:
//
//	log.Errorf("the number is %d", 123)
func (l *Logger) Errorf(format string, args ...any) identifier.ID {
	return l.inst.log(ERR, format, args, true, 3, l.fieldData, l.fieldCount)
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
