package fluentlog

type Severity uint8

const (
	EMERG Severity = iota
	ALERT
	CRIT
	ERR
	WARNING
	NOTICE
	INFO
	DEBUG
)
