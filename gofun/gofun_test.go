package gofun

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	c := context.Background()
	fn := func(i int) (err error) {
		time.Sleep(1 * time.Second)
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
		err = errors.New("aaaaaaa")
		return
	}
	fmt.Println(runtime.Caller(0))

	log := &DefaultLogger{}
	gofn := NewGoFunc(WithLogger(log))

	//gofn.Go(c, fn)

	gofn.WithTimeout(
		c,
		time.Second*4,
		fn,
		fn,
		fn,
		fn,
		fn,
		fn,
	)
	fmt.Println(fmt.Sprintf("%+v", log.Err()))
}
