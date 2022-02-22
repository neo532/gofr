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
)

// guardpanic is a statement of guardpanic.
type guardpanic struct {
	restartTimes  int
	workerFn      func()
	errCallBackFn func(c context.Context, err error)
	ctx           context.Context
}

// Recover is a method for panic.
func (gp *guardpanic) Recover() {

	if r := recover(); r != nil {

		if gp.errCallBackFn != nil {
			gp.errCallBackFn(
				gp.ctx,
				fmt.Errorf("%s, %s",
					r,
					string(debug.Stack()),
				),
			)
		}

		if gp.restartTimes > 0 {

			gp.restartTimes--
			go Run(
				gp.ctx,
				gp.workerFn,
				gp.restartTimes,
				gp.errCallBackFn,
			)
		}
	}
}

// Run is a function for goroutine.
func Run(c context.Context, worker func(), times int, cb func(c context.Context, err error)) {

	gp := &guardpanic{
		workerFn:      worker,
		restartTimes:  times,
		errCallBackFn: cb,
		ctx:           c,
	}
	defer gp.Recover()

	go worker()
	return
}
