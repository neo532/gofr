package tool

/*
 * @abstract lock for multi-server
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"time"
)

// ILockDb is the interface for Lock's db.
type ILockDb interface {
	Incr(key string) (int64, error)
	Expire(key string, expiration time.Duration) (bool, error)
	Del(key string) (int64, error)
}

// Lock is the instance for Lock.
type Lock struct {
	db ILockDb
}

// NewLock returns the instance for Lock.
func NewLock(d ILockDb) *Lock {
	return &Lock{
		db: d,
	}
}

// Lock locks and returns the result if locking is successfully.
// sec[0] : expireSec
// sec[1] : waitSec
func (l *Lock) Lock(key string, sec ...int) bool {
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

	key = l.getKey(key)
	endTs := time.Now().Add(time.Duration(waitSec) * time.Second)
	for {
		if num, err := l.db.Incr(key); nil == err && num == 1 {
			l.db.Expire(key, time.Duration(expireSec)*time.Second)
			return true
		}

		if time.Now().After(endTs) {
			break
		}

		time.Sleep(time.Duration(50) * time.Millisecond)
	}
	return false
}

// UnLock unlocks.
func (l *Lock) UnLock(key string) (int64, error) {
	return l.db.Del(l.getKey(key))
}

func (l *Lock) getKey(key string) string {
	return "lock:" + key
}
