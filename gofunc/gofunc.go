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
	taskStatusEnd = -1
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

	go func() {

		defer func() {
			if r := recover(); r != nil {
				g.log.Error(c,
					errors.Errorf("[producer][%+v][%s]", r, string(debug.Stack())),
				)
			}
			for i := 0; i < lRunning; i++ {
				task <- taskStatusEnd
			}
			wg.Done()
			close(task)
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
			var index int
			defer func() {
				if r := recover(); r != nil {

					g.log.Error(c,
						errors.Errorf("[%dth][%+v][%s]", index, r, string(debug.Stack())),
					)
					// pop taskStatusEnd
					<-task
				}
				wg.Done()
			}()

			for {
				if index = <-task; index == taskStatusEnd {
					return
				}

				if err := fns[index](index); err != nil {
					g.log.Error(c, errors.Wrapf(err, "[%dth]", index))
				}
			}
		}()
	}

	wg.Wait()
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
