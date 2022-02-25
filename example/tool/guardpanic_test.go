package tool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/neo532/gofr/tool"
)

func TestGuardpanic(t *testing.T) {
	c := context.Background()
	fn := func(i int) {
		time.Sleep(2 * time.Second)
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
	}

	var opt tool.GFopt
	gofn := tool.NewGoFunc(opt.ErrFunc(logger.CErr))

	//gofn.Go(c, fn)

	gofn.WithTimeout(c, time.Second*3, fn)
}
