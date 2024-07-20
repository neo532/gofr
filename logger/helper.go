package logger

import (
	"context"
	"fmt"
)

type Logger struct {
	log        ILogger
	globalArgs []interface{}
}

func NewLogger(l ILogger) *Logger {
	return &Logger{
		log:        l,
		globalArgs: make([]interface{}, 0, 1),
	}
}

func (l *Logger) WithArgs(kvs ...interface{}) (n *Logger) {
	n = NewLogger(l.log)
	n.globalArgs = kvs
	return n
}

func (l *Logger) Debugf(c context.Context, format string, kvs ...interface{}) {
	l.log.Log(c, LevelDebug, fmt.Sprintf(format, kvs...), l.globalArgs...)
}
func (l *Logger) Warnf(c context.Context, format string, kvs ...interface{}) {
	l.log.Log(c, LevelWarn, fmt.Sprintf(format, kvs...), l.globalArgs...)
}
func (l *Logger) Infof(c context.Context, format string, kvs ...interface{}) {
	l.log.Log(c, LevelInfo, fmt.Sprintf(format, kvs...), l.globalArgs...)
}
func (l *Logger) Errorf(c context.Context, format string, kvs ...interface{}) {
	l.log.Log(c, LevelError, fmt.Sprintf(format, kvs...), l.globalArgs...)
}
func (l *Logger) Fatalf(c context.Context, format string, kvs ...interface{}) {
	l.log.Log(c, LevelFatal, fmt.Sprintf(format, kvs...), l.globalArgs...)
}

func (l *Logger) Debug(c context.Context, message string, kvs ...interface{}) {
	l.log.Log(c, LevelDebug, message, append(l.globalArgs, kvs...)...)
}
func (l *Logger) Warn(c context.Context, message string, kvs ...interface{}) {
	l.log.Log(c, LevelWarn, message, append(l.globalArgs, kvs...)...)
}
func (l *Logger) Info(c context.Context, message string, kvs ...interface{}) {
	l.log.Log(c, LevelInfo, message, append(l.globalArgs, kvs...)...)
}
func (l *Logger) Error(c context.Context, message string, kvs ...interface{}) {
	l.log.Log(c, LevelError, message, append(l.globalArgs, kvs...)...)
}
func (l *Logger) Fatal(c context.Context, message string, kvs ...interface{}) {
	l.log.Log(c, LevelFatal, message, append(l.globalArgs, kvs...)...)
}
