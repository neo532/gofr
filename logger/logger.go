package logger

import (
	"context"
)

type ILogger interface {
	Log(c context.Context, level Level, message string, kvs ...interface{}) error
	Close() error
}

type ILoggerArgs func(c context.Context) (k string, v interface{})

type Logger interface {
	WithArgs(kvs ...interface{}) (n Logger)
	WithLevel(lv Level) (n Logger)
	Close() error

	Debugf(c context.Context, format string, kvs ...interface{})
	Warnf(c context.Context, format string, kvs ...interface{})
	Infof(c context.Context, format string, kvs ...interface{})
	Errorf(c context.Context, format string, kvs ...interface{})
	Fatalf(c context.Context, format string, kvs ...interface{})

	Debug(c context.Context, message string, kvs ...interface{})
	Warn(c context.Context, message string, kvs ...interface{})
	Info(c context.Context, message string, kvs ...interface{})
	Error(c context.Context, message string, kvs ...interface{})
	Fatal(c context.Context, message string, kvs ...interface{})
}
