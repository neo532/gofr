package orm

import (
	"context"
	"fmt"
)

/*
 * @abstract Orm's Logger
 * @mail neo532@126.com
 * @date 2024-05-18
 */

type Logger interface {
	Error(c context.Context, message string, kvs ...interface{})
	Warn(c context.Context, message string, kvs ...interface{})
	Info(c context.Context, message string, kvs ...interface{})
}

type DefaultLogger struct {
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}
func (l *DefaultLogger) Error(c context.Context, message string, kvs ...interface{}) {
	fmt.Println(append([]interface{}{"msg", message}, kvs...)...)
}
func (l *DefaultLogger) Warn(c context.Context, message string, kvs ...interface{}) {
	fmt.Println(append([]interface{}{"msg", message}, kvs...)...)
}
func (l *DefaultLogger) Info(c context.Context, message string, kvs ...interface{}) {
	fmt.Println(append([]interface{}{"msg", message}, kvs...)...)
}
