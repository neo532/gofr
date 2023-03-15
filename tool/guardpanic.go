package tool

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
	errFn   func(c context.Context, err error)
}

// GFopt is a object for guard goroutine and panic.
type GFopt func(*GoFunc)

// ErrFunc sets the handle of error for GoFunc.
func (l GFopt) ErrFunc(fn func(c context.Context, err error)) GFopt {
	return func(v *GoFunc) {
		v.errFn = fn
	}
}

// NewGoFunc returns a instance of GoFunc.
func NewGoFunc(opts ...GFopt) *GoFunc {
	gf := &GoFunc{
		errFn: defErrFn,
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
	for i := range fns {

		go func(j int) {
			defer wg.Done()
			finish := make(chan struct{}, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						g.errFn(
							c,
							fmt.Errorf("[%d][%s][%s]", j, r, string(debug.Stack())),
						)
					}
				}()
				fns[i](j)
				finish <- struct{}{}
			}()

			for {
				select {
				case <-time.After(ts):
					g.errFn(c, fmt.Errorf("[%d]timeout", j))
					return
				case <-finish:
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
	for i := range fns {

		go func(j int) {
			defer func() {
				if r := recover(); r != nil {
					g.errFn(
						c,
						fmt.Errorf("[%d][%s][%s]", j, r, string(debug.Stack())),
					)
				}
			}()
			defer wg.Done()
			fn[j](j)
		}(i)
	}
	wg.Wait()
}

// AsyncWithTimeout is a way that running groutine slice by limiting time is asynchronized.
func (g *GoFunc) AsyncWithTimeout(c context.Context, ts time.Duration, fns ...func(i int)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.errFn(
					c,
					fmt.Errorf("[%s][%s]", r, string(debug.Stack())),
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
				g.errFn(
					c,
					fmt.Errorf("[%s][%s]", r, string(debug.Stack())),
				)
			}
		}()
		g.Go(c, fns...)
	}()
}
