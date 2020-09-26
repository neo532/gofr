/*
 * @abstract lock for multi-server
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package tool

import (
	"time"
)

type LockDb interface {
	Incr(key string) (int64, error)
	Expire(key string, expiration time.Duration)
	Del(key string) (int64 error)
}

type Lock struct {
	db LockDb
}

func NewLock(d LockDb) *Lock {
	return &Lock{
		db: d,
	}
}

//sec[0] : expireSec
//sec[1] : waitSec
func (this *Tool) Lock(key string, sec ...int) bool {
	expireSec := 0
	waitSec := 0
	switch len(sec) {
	case 1:
		expireSec = sec[0]
		waitSec = expireSec
	case 2:
		expireSec = sec[0]
		waitSec = sec[1]
	default:
		return false
	}

	key = this.getKey(key)
	endTs := time.Now().Add(time.Duration(waitSec) * time.Second)
	for {
		if time.Now().After(endTs) {
			break
		}

		if num, err := this.db.Incr(key); nil == err && num == 1 {
			this.db.Expire(key, time.Duration(expireSec)*time.Second)
			return true
		}

		time.Sleep(time.Duration(50) * time.Millisecond)
	}
	return false
}

func (this *Lock) UnLock(key string) (int64, error) {
	return this.db.Del(this.getKey(key))
}

func (this *Lock) getKey(key string) string {
	return "lock:" + key
}
