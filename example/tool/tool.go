package tool

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
)

var (
	rdb    *rDb
	logger *log
)

func init() {
	rdb = newRdb()
	logger = &log{}
}

// ========== rDb ==========
type rDb struct {
	cache *redis.Client
}

func (l *rDb) Eval(c context.Context, cmd string, keys []string, args []interface{}) (rst interface{}, err error) {
	return l.cache.Eval(cmd, keys, args...).Result()
}
func (l *rDb) Get(c context.Context, key string) (rst string, err error) {
	rst, err = l.cache.Get(key).Result()
	return
}

func newRdb() *rDb {
	return &rDb{redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})}
}

// ========== log ==========
type log struct {
}

func (l *log) CErr(c context.Context, err error) {
	fmt.Println(err)
}
