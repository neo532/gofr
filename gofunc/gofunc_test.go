package gofunc

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	c, closeFn := context.WithCancel(context.Background())
	fn := func(i int) (err error) {
		time.Sleep(time.Second * 1)
		fmt.Println(runtime.Caller(0))
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
		//err = errors.New("aaaaaaa")
		return
	}

	log := &DefaultLogger{}
	gofn := NewGoFunc(WithLogger(log), WithMaxGoroutine(20))

	//gofn.Go(c, fn, fn, fn)

	go func() {
		go func() {
			gofn.WithTimeout(
				c,
				time.Second*5,
				fn,
				fn,
			)
			fmt.Println(fmt.Sprintf("print<<<\t%+v>>>", log.Err()))
		}()
		select {
		case <-c.Done():
			return
		}
		fmt.Println(runtime.Caller(0))
	}()

	time.Sleep(4 * time.Second)
	fmt.Println(runtime.Caller(0))
	closeFn()
}
