package tool

/*
 * @abstract lock for multi-server in one redis instance
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-05
 * @demo:

	"github.com/go-redis/redis"

	type lockDb struct {
		cache *redis.Client
	}

	func (l *lockDb) Eval(c context.Context, cmd string, keys []string, args []interface{}) (err string) {
		err, _ = l.cache.Eval(cmd, keys, args...).String()
		return
	}
	var c = context.Background()
	rdb := &lockDb{redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "password",
	})}
	var l = tool.NewLock(rdb)
	code, err := l.Lock(
		c,
		"key1",
		time.Duration(1) * time.Second,
		time.Duration(1) * time.Second,
	)
	l.UnLock(c, "key1", code)
*/

import (
	"context"
	"errors"
	"time"

	uuid "github.com/satori/go.uuid"
)

const EVAL_OK = "ok"

// args:1 keyName code 10
var lockLuaScript = `
local key=KEYS[1] 
local code=ARGV[1]
local expire=ARGV[2]
local rst=redis.call('SET', key, code, 'EX', expire, 'NX')
if(rst==false) then
	return 'set fail'
end
return '` + EVAL_OK + `'
`

// args:1 keyName code
var unlockLuaScript = `
local key=KEYS[1]
local code=ARGV[1] 
local value=redis.call('GET', key)
if(value==false) then
	return 'get fail'
end 
if(code~=value) then
	return 'equal fail' 
end
local rst=redis.call('DEL', key)
if(rst==0) then
	return 'del fail' 
end
return '` + EVAL_OK + `'
`

// ILockDb is the interface for Lock's db.
type ILockDb interface {
	Eval(c context.Context, cmd string, keys []string, args []interface{}) (err string)
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

// UnLock unlocks.
func (l *Lock) UnLock(c context.Context, key string, code string) (err error) {
	key = getLockKey(key)
	e := l.db.Eval(c, unlockLuaScript, []string{key}, []interface{}{code})
	if e == EVAL_OK {
		err = nil
		return
	}
	err = errors.New(e)
	return
}

// Lock locks and returns the result if locking is successfully.
func (l *Lock) Lock(c context.Context, key string, expire, wait time.Duration) (code string, err error) {
	code = uuid.NewV4().String()
	key = getLockKey(key)

	endTs := time.Now().Add(wait)
	for time.Now().Before(endTs) {

		e := l.db.Eval(c, lockLuaScript, []string{key}, []interface{}{code, expire.Seconds()})
		if e == EVAL_OK {
			err = nil
			return
		}
		err = errors.New(e)

		time.Sleep(time.Duration(50) * time.Millisecond)
	}

	return
}

func getLockKey(key string) string {
	return "lock:" + key
}
