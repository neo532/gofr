package gofunc

/*
 * @abstract guard panic
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-06
 */

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	TaskStatusEnd      = -1
	TaskStatusTimeout  = -2
	TaskStatusProducer = -3
)

// GoFunc is a function for a goroutine.
type GoFunc struct {
	timeout      time.Duration
	log          Logger
	maxGoroutine int
}

// opt is a object for guard goroutine and panic.
type opt func(*GoFunc)

// WithLogger sets the handle of error for GoFunc.
func WithLogger(log Logger) opt {
	return func(v *GoFunc) {
		v.log = log
	}
}

// WithMaxGoroutine sets the limit of goroutine at the same time.
func WithMaxGoroutine(n int) opt {
	return func(v *GoFunc) {
		v.maxGoroutine = n
	}
}

// NewGoFunc returns a instance of GoFunc.
func NewGoFunc(opts ...opt) *GoFunc {
	gf := &GoFunc{
		log:          &DefaultLogger{},
		maxGoroutine: 10,
	}
	for _, o := range opts {
		o(gf)
	}
	return gf
}

func (g *GoFunc) goWithTimeout(c context.Context, ts time.Duration, fns ...func(i int) error) {

	l := len(fns)
	lRunning := g.maxGoroutine
	if l < lRunning {
		lRunning = l
	}

	var wg sync.WaitGroup
	wg.Add(lRunning + 1)
	task := make(chan int)

	defer func() {
		wg.Wait()
		close(task)
	}()

	go func() {
		taskStatus := TaskStatusEnd

		defer func() {
			if r := recover(); r != nil {
				taskStatus = TaskStatusProducer
				g.log.Error(c,
					errors.Errorf("[%dth][%+v][%s]", taskStatus, r, string(debug.Stack())),
				)
			}
			for j := 0; j < lRunning; j++ {
				task <- taskStatus
			}
			wg.Done()
		}()

		if int(ts.Microseconds()) == 0 {
			for i := 0; i < l; i++ {
				task <- i
			}
			return
		}

		timer := time.NewTimer(ts)
		defer timer.Stop()

		for i := 0; i < l; i++ {
			select {
			case <-timer.C:
				taskStatus = TaskStatusTimeout
				g.log.Error(c,
					errors.Errorf("Timeout!,goroutines faild to finish within the specified %v", ts),
				)
				return
			default:
				task <- i
			}
		}
	}()

	for i := 0; i < lRunning; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case index := <-task:
					switch index {
					case TaskStatusTimeout, TaskStatusProducer, TaskStatusEnd:
						return
					}
					defer func() {
						if r := recover(); r != nil {
							g.log.Error(c,
								errors.Errorf("[%dth][%+v][%s]", index, r, string(debug.Stack())),
							)
						}
					}()
					fns[index](index)
				}
			}
		}()
	}
}

// WithTimeout is a way that running groutine slice by limiting time is synchronized.
func (g *GoFunc) WithTimeout(c context.Context, ts time.Duration, fns ...func(i int) error) {
	g.goWithTimeout(c, ts, fns...)
}

// Go is a way that running groutine slice is synchronized.
func (g *GoFunc) Go(c context.Context, fns ...func(i int) error) {
	g.goWithTimeout(c, time.Second*0, fns...)
}

// AsyncWithTimeout is a way that running groutine slice by limiting time is asynchronized.
func (g *GoFunc) AsyncWithTimeout(c context.Context, ts time.Duration, fns ...func(i int) error) {
	go func() {
		g.WithTimeout(c, ts, fns...)
	}()
}

// AsyncGo is a way that running groutine slice is asynchronized.
func (g *GoFunc) AsyncGo(c context.Context, fns ...func(i int) error) {
	go func() {
		g.Go(c, fns...)
	}()
}
