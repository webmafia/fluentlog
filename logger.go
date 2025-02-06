package fluentlog

import (
	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/identifier"
)

type Logger struct {
	inst       *Instance
	fieldData  *buffer.Buffer
	fieldCount uint8
}

func NewLogger(inst *Instance) *Logger {
	return inst.Logger()
}

func (l *Logger) Debug(msg string, args ...any) identifier.ID {
	return l.inst.log(DEBUG, msg, args, l.fieldData.B, l.fieldCount)
}
func (l *Logger) Info(msg string, args ...any) identifier.ID {
	return l.inst.log(INFO, msg, args, l.fieldData.B, l.fieldCount)
}
func (l *Logger) Warn(msg string, args ...any) identifier.ID {
	return l.inst.log(WARN, msg, args, l.fieldData.B, l.fieldCount)
}
func (l *Logger) Error(msg string, args ...any) identifier.ID {
	return l.inst.log(ERR, msg, args, l.fieldData.B, l.fieldCount)
}

// With implements Instance.
func (l *Logger) With(args ...any) *Logger {
	log := l.inst.Logger()
	log.fieldData.B = append(log.fieldData.B, l.fieldData.B...)
	log.fieldCount = l.fieldCount + appendArgs(log.fieldData, args)

	return log
}

// Release implements Instance.
func (l *Logger) Release() {
	l.inst.Release(l)
}
