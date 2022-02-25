package tool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/neo532/gofr/tool"
)

var (
	user  *tool.VarStorageByLock
	money *tool.VarStorageByTick
)

type UserData struct {
}

func (l *UserData) IsVaild(c context.Context) bool {
	t, err := rdb.cache.TTL("userkey").Result()
	if err == nil && int64(t.Seconds()) == 0 {
		return false
	}
	return true
}
func (l *UserData) Update(c context.Context) interface{} {
	rdb.cache.SetNX("userkey", "zhangsan", time.Second*2)
	return "zhangsan"
}

func init() {
	var uO tool.VSLopt
	user = tool.NewVarStorageByLock(
		uO.OpFun(&UserData{}),
		uO.ErrFun(logger.CErr),
	)

	var mO tool.VSTopt
	money = tool.NewVarStorageByTick(
		mO.OpFun(&UserData{}),
		mO.ErrFun(logger.CErr),
	)
}

func TestVarStorageByLock(t *testing.T) {
	c := context.Background()
	d := user.Get(c)
	fmt.Println(fmt.Sprintf("%s\t:get,%v", t.Name(), d))

	time.Sleep(time.Second * 3)
}

func TestVarStorageByTick(t *testing.T) {
	d := money.Get()
	fmt.Println(fmt.Sprintf("%s\t:get,%v", t.Name(), d))
}
