package gofun

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	c := context.Background()
	fn := func(i int) {
		time.Sleep(3 * time.Second)
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
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
