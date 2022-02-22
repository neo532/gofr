package tool

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/neo532/gofr/tool"
)

var (
	distributedLock *tool.DistributedLock
)

func init() {
	distributedLock = tool.NewDistributedLock(rdb)
}

func TestDistributedLock(t *testing.T) {

	var code string
	var err error

	c := context.Background()
	key := "IamAKey"
	expire := time.Duration(10) * time.Second
	wait := time.Duration(2) * time.Second

	count := 10000
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			var ct string
			ct, err = distributedLock.Lock(c, key, expire, wait)
			if err == nil {
				code = ct
				fmt.Println(fmt.Sprintf("%s\t:Biz run!", t.Name()))
			}
		}()
	}
	wg.Wait()

	distributedLock.UnLock(c, key, code)
}
