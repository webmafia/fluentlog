package fluentlog

import "github.com/webmafia/identifier"

type Instance interface {
	With(args ...any) *SubLogger

	Debug(msg string, args ...any) identifier.ID
	Info(msg string, args ...any) identifier.ID
	Warn(msg string, args ...any) identifier.ID
	Error(msg string, args ...any) identifier.ID
}
