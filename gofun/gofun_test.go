package gofun

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	c := context.Background()
	fn := func(i int) (err error) {
		time.Sleep(2 * time.Second)
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
		//err = errors.New("aaaaaaa")
		return
	}

	log := &DefaultLogger{}
	gofn := NewGoFunc(WithLogger(log))

	gofn.Go(c, fn, fn, fn)

	gofn.WithTimeout(
		c,
		time.Second*3,
		fn,
		fn,
		fn,
		fn,
		fn,
		fn,
	)
	fmt.Println(fmt.Sprintf("%+v", log.Err()))
}
