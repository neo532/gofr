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

type GoFunc struct {
	timeout time.Duration
	errFn   func(c context.Context, err error)
}
type GFopt func(*GoFunc)

// ErrFunc sets the handle of error for GoFunc.
func (l GFopt) ErrFunc(fn func(c context.Context, err error)) GFopt {
	return func(v *GoFunc) {
		v.errFn = fn
	}
}

func NewGoFunc(opts ...GFopt) *GoFunc {
	gf := &GoFunc{
		errFn: defErrFn,
	}
	for _, o := range opts {
		o(gf)
	}
	return gf
}

func (g *GoFunc) WithTimeout(c context.Context, ts time.Duration, fns ...func(i int)) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for i, fn := range fns {

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
				fn(j)
				finish <- struct{}{}
			}()

			for {
				select {
				case <-time.After(ts):
					fmt.Println("timeout")
					return
				case <-finish:
					fmt.Println("finish")
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func (g *GoFunc) Go(c context.Context, fns ...func(i int)) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for i, fn := range fns {

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
			fn(j)
		}(i)
	}
	wg.Wait()
}

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
