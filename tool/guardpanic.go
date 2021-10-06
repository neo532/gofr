package tool

/*
 * @abstract guard panic
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-06
 * @demo:
 */

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
)

// guardpanic is a statement of guardpanic.
type guardpanic struct {
	restartTimes  int
	workerFn      func()
	errCallBackFn func(err error)
}

// Recover is a method for panic.
func (gp *guardpanic) Recover() {

	if r := recover(); r != nil {

		err := errors.New(
			fmt.Sprintf("%s, %s", r, string(debug.Stack())),
		)

		if gp.errCallBackFn != nil {
			gp.errCallBackFn(err)
		} else {
			fmt.Fprint(os.Stderr, err)
		}

		if gp.restartTimes > 0 {

			gp.restartTimes--
			go Run(
				gp.workerFn,
				gp.restartTimes,
				gp.errCallBackFn,
			)
		}
	}
}

// Run is a function for goroutine.
func Run(worker func(), times int, cb func(err error)) {

	gp := &guardpanic{
		workerFn:      worker,
		restartTimes:  times,
		errCallBackFn: cb,
	}
	defer gp.Recover()

	worker()
	return
}
