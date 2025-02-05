package fluentlog

import (
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/identifier"
)

var _ Instance = (*SubLogger)(nil)

type SubLogger struct {
	base       *Logger
	fieldData  *buffer.Buffer
	fieldCount uint8
}

func (l *SubLogger) Debug(msg string, args ...any) identifier.ID {
	return l.base.log(DEBUG, msg, args, l.fieldData.B, l.fieldCount)
}
func (l *SubLogger) Info(msg string, args ...any) identifier.ID {
	return l.base.log(INFO, msg, args, l.fieldData.B, l.fieldCount)
}
func (l *SubLogger) Warn(msg string, args ...any) identifier.ID {
	return l.base.log(WARN, msg, args, l.fieldData.B, l.fieldCount)
}
func (l *SubLogger) Error(msg string, args ...any) identifier.ID {
	return l.base.log(ERR, msg, args, l.fieldData.B, l.fieldCount)
}

// With implements Instance.
func (l *SubLogger) With(args ...any) *SubLogger {
	subLog, ok := l.base.subPool.Get().(*SubLogger)

	if !ok {
		subLog = &SubLogger{
			base: l.base,
		}
	}

	subLog.fieldData = l.base.pool.Get()
	subLog.fieldData.B = append(subLog.fieldData.B, l.fieldData.B...)
	subLog.fieldCount = l.fieldCount + appendArgs(subLog.fieldData, args)

	return subLog
}

// Release implements Instance.
func (l *SubLogger) Release() {
	l.base.pool.Put(l.fieldData)
	l.fieldData = nil
	l.fieldCount = 0
	l.base.subPool.Put(l)
}
