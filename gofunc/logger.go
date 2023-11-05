package gofunc

/*
 * @abstract guard panic
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-06
 */

import (
	"context"
	"sync"

	"github.com/pkg/errors"
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
	defer l.lock.Unlock()
	if l.err != nil {
		l.err = errors.Wrap(l.err, message)
		return
	}
	l.err = errors.New(message)
}

func (l *DefaultLogger) Info(c context.Context, message string) {
}

func (l *DefaultLogger) Err() error {
	return l.err
}
