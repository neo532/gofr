package tool

import (
	"context"
	"fmt"
	"time"
)

var (
	defDuration = time.Second
	defTimeout  = time.Second * 3
	defRetry    = 1
	defErrFn    = func(c context.Context, err error) {
		fmt.Println(fmt.Sprintf("has error[%v]", err))
	}
	defFn = func() {
	}
)
