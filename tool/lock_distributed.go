package tool

/*
 * @abstract lock for multi-server in one redis instance
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-05
 * @example: github.com/neo532/gofr/blob/main/example/tool/lock_distributed_test.go
 */

import (
	"context"
	"errors"
	"time"

	uuid "github.com/satori/go.uuid"
)

const evalOk = "ok"

// args:1 keyName code 10
var lockLuaScript = `
local key=KEYS[1] 
local code=ARGV[1]
local expire=ARGV[2]
local rst=redis.call('SET', key, code, 'EX', expire, 'NX')
if(rst==false) then
	return 'set fail'
end
return '` + evalOk + `'
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
return '` + evalOk + `'
`

// IDistributedLockDb is the interface for DistributedLock's db.
type IDistributedLockDb interface {
	Eval(c context.Context, cmd string, keys []string, args []interface{}) (rst interface{}, err error)
}

// DistributedLock is the instance for DistributedLock.
type DistributedLock struct {
	db       IDistributedLockDb
	duration time.Duration
	genCode  func() (s string, err error)
}

// NewDistributedLock returns the instance for Lock.
func NewDistributedLock(d IDistributedLockDb) *DistributedLock {
	return &DistributedLock{
		db:       d,
		duration: time.Duration(50) * time.Millisecond,
		genCode: func() (s string, err error) {
			s = uuid.NewV4().String()
			return
		},
	}
}

// GenCodeFun returns a unique code.
func (l *DistributedLock) GenUniqCodeFn(fn func() (s string, err error)) *DistributedLock {
	l.genCode = fn
	return l
}

// Duration sets the duration on lock.
func (l *DistributedLock) Duration(d time.Duration) *DistributedLock {
	l.duration = d
	return l
}

// UnLock unlocks.
func (l *DistributedLock) UnLock(c context.Context, key string, code string) (err error) {
	key = getLockKey(key)
	var rst interface{}
	if rst, err = l.db.Eval(c, unlockLuaScript, []string{key}, []interface{}{code}); err != nil {
		return
	}
	e, ok := rst.(string)
	if ok && e == evalOk {
		err = nil
		return
	}
	err = errors.New(e)
	return
}

// Lock locks and returns the result if locking is successfully.
func (l *DistributedLock) Lock(c context.Context, key string, expire, wait time.Duration) (code string, err error) {
	code, err = l.genCode()
	if err != nil {
		return
	}
	key = getLockKey(key)

	endTs := time.Now().Add(wait)
	for time.Now().Before(endTs) {
		var rst interface{}
		if rst, err = l.db.Eval(c, lockLuaScript, []string{key}, []interface{}{code, expire.Seconds()}); err != nil {
			return
		}

		e, ok := rst.(string)
		if ok && e == evalOk {
			err = nil
			return
		}
		err = errors.New(e)

		time.Sleep(l.duration)
	}

	if err == nil {
		err = errors.New("timeout")
	}
	return
}

func getLockKey(key string) string {
	return "lock:" + key
}
