package tool

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/neo532/gofr/tool"
)

var (
	freq *tool.Freq
)

func init() {
	freq = tool.NewFreq(rdb)
	freq.Timezone("Local")
}

func TestFreq(t *testing.T) {

	c := context.Background()
	preKey := "user.test"
	rule := []tool.FreqRule{
		tool.FreqRule{Duri: "10", Times: 2},
	}

	count := 10000
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			ok, err := freq.IncrCheck(c, preKey, rule...)
			if err == nil && ok {
				fmt.Println(fmt.Sprintf("%s\t:Biz run!", t.Name()))
			}
		}()
	}
	wg.Wait()

	var err error

	var b bool
	b, err = freq.IncrCheck(c, preKey, rule...)
	fmt.Println(fmt.Sprintf("%s\t:incrCheck!,%v,%v", t.Name(), b, err))

	var times int64
	times, err = freq.Get(c, preKey, rule...)
	fmt.Println(fmt.Sprintf("%s\t:freqGet!,%d,%v", t.Name(), times, err))
}
