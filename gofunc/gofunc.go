package gofunc

/*
 * @abstract guard panic
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-06
 */

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pkg/errors"
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

	running := make(chan int, lRunning)
	defer close(running)

	finish := make(chan int, l)
	defer close(finish)

	go func() {
		var wg sync.WaitGroup
		wg.Add(l)
		for i := 0; i < l; i++ {
			running <- i
			go func(j int) {
				defer func() {
					wg.Done()
					finish <- j
					if r := recover(); r != nil {
						g.log.Error(
							c,
							errors.Errorf("[%dth][%+v][%s]", j, r, string(debug.Stack())),
						)
					}
				}()
				if err := fns[j](j); err != nil {
					g.log.Error(c, errors.Wrapf(err, "[%dth]", j))
				}
			}(i)
		}
		wg.Wait()
		finish <- -1
	}()

	if int(ts.Microseconds()) == 0 {
		for {
			select {
			case n := <-finish:
				if n == -1 {
					return
				} else {
					<-running
				}
				g.log.Info(c, fmt.Sprintf("Finish %dth", n))
			}
		}
		return
	}

	for {
		select {
		case <-time.After(ts):
			g.log.Error(c, errors.Errorf("Timeout!,goroutines faild to finish within the specified %v", ts))
			return
		case n := <-finish:
			if n == -1 {
				return
			} else {
				<-running
			}
			g.log.Info(c, fmt.Sprintf("Finish %dth", n))
		}
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
