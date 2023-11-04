package gofun

/*
 * @abstract guard panic
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-06
 */

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Logger interface {
	Error(c context.Context, message string)
	Info(c context.Context, message string)
}

type DefaultLogger struct {
	err  error
	lock sync.Mutex
}

func (l *DefaultLogger) Error(c context.Context, message string) {
	l.lock.Lock()
	l.err = errors.New(message)
	l.lock.Unlock()
}

func (l *DefaultLogger) Info(c context.Context, message string) {
	fmt.Println(fmt.Sprintf("%+v", message))
}

func (l *DefaultLogger) Err() error {
	return l.err
}
