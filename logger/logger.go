package logger

import (
	"context"
)

type ILogger interface {
	Log(c context.Context, level Level, message string, kvs ...interface{}) error
}

type ILoggerArgs func(c context.Context) (k string, v interface{})
