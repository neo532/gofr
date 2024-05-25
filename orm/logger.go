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
	Errorf(c context.Context, format string, p ...interface{})
	Warnf(c context.Context, format string, p ...interface{})
	Infof(c context.Context, format string, p ...interface{})
}

type DefaultLogger struct {
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}
func (l *DefaultLogger) Errorf(c context.Context, format string, p ...interface{}) {
	fmt.Println(fmt.Sprintf(format, p...))
}
func (l *DefaultLogger) Warnf(c context.Context, format string, p ...interface{}) {
	fmt.Println(fmt.Sprintf(format, p...))
}
func (l *DefaultLogger) Infof(c context.Context, format string, p ...interface{}) {
	fmt.Println(fmt.Sprintf(format, p...))
}
