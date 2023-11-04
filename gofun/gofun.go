package gofun

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
)

// GoFunc is a function for a goroutine.
type GoFunc struct {
	timeout time.Duration
	log     Logger
}

// opt is a object for guard goroutine and panic.
type opt func(*GoFunc)

// WithLogger sets the handle of error for GoFunc.
func WithLogger(log Logger) opt {
	return func(v *GoFunc) {
		v.log = log
	}
}

// NewGoFunc returns a instance of GoFunc.
func NewGoFunc(opts ...opt) *GoFunc {
	gf := &GoFunc{
		log: &DefaultLogger{},
	}
	for _, o := range opts {
		o(gf)
	}
	return gf
}

// WithTimeout is a way that running groutine slice by limiting time is synchronized.
func (g *GoFunc) WithTimeout(c context.Context, ts time.Duration, fns ...func(i int)) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for i, fn := range fns {

		go func(j int) {
			defer wg.Done()
			finish := make(chan int, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						g.log.Error(
							c,
							fmt.Sprintf("[%d][%s][%s]", j, r, string(debug.Stack())),
						)
					}
				}()
				fn(j)
				finish <- j
			}()

			for {
				select {
				case <-time.After(ts):
					g.log.Error(c, fmt.Sprintf("Timeout!,goroutines faild to finish within the specified %v", ts))
					return
				case n := <-finish:
					g.log.Info(c, fmt.Sprintf("Finish %dth", n))
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

// Go is a way that running groutine slice is synchronized.
func (g *GoFunc) Go(c context.Context, fns ...func(i int)) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for i, fn := range fns {

		go func(j int) {
			defer func() {
				if r := recover(); r != nil {
					g.log.Error(
						c,
						fmt.Sprintf("[%d][%s][%s]", j, r, string(debug.Stack())),
					)
				}
			}()
			defer wg.Done()
			fn(j)
		}(i)
	}
	wg.Wait()
}

// AsyncWithTimeout is a way that running groutine slice by limiting time is asynchronized.
func (g *GoFunc) AsyncWithTimeout(c context.Context, ts time.Duration, fns ...func(i int)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.log.Error(
					c,
					fmt.Sprintf("[%s][%s]", r, string(debug.Stack())),
				)
			}
		}()
		g.WithTimeout(c, ts, fns...)
	}()
}

// AsyncGo is a way that running groutine slice is asynchronized.
func (g *GoFunc) AsyncGo(c context.Context, fns ...func(i int)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.log.Error(
					c,
					fmt.Sprintf("[%s][%s]", r, string(debug.Stack())),
				)
			}
		}()
		g.Go(c, fns...)
	}()
}
