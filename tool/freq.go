package tool

/*
 * @abstract frequency control
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 * @example: github.com/neo532/gofr/blob/main/example/tool/freq_test.go
 */

import (
	"context"
	"strconv"
	"time"

	"github.com/neo532/gofr/lib"
)

// args:1 keyName 10
var incrLuaScript = `
local key=KEYS[1]
local expire=ARGV[1]
local incr=redis.call('INCR', key)
if(incr~=1) then
return incr
end
local rst=redis.call('EXPIRE', key, expire)
if(rst~=1) then
return -1
end
return incr
`

// IFreqDb is the interface for FreqRule.
type IFreqDb interface {
	Eval(c context.Context, cmd string, keys []string, args []interface{}) (rst interface{}, err error)
	Get(c context.Context, key string) (string, error)
}

// FreqRule is the instance for FreqRule.
type FreqRule struct {
	Duri  string //3|day
	Times int64
}

// Freq is the instance for FreqRule.
type Freq struct {
	tz *time.Location
	db IFreqDb
}

// NewFreq returns a instance of Freq.
func NewFreq(d IFreqDb) *Freq {
	return &Freq{
		db: d,
	}
}

// Timezone sets the timezone for the day in FreqRule.
func (f *Freq) Timezone(timezone string) (err error) {
	f.tz, err = time.LoadLocation(timezone)
	return
}

// Get return the last count.
func (f *Freq) Get(c context.Context, pre string, rule ...FreqRule) (ts int64, err error) {
	f.freq(pre, rule, func(key string, expire, times int64) bool {
		var tsOri string
		if tsOri, err = f.db.Get(c, key); err != nil {
			return false
		}

		ts, err = strconv.ParseInt(tsOri, 10, 64)
		return true
	})
	return
}

// Check checks the count only.
func (f *Freq) Check(c context.Context, pre string, rule ...FreqRule) (bRst bool, err error) {
	bRst = f.freq(pre, rule, func(key string, expire, times int64) bool {
		var tsOri string
		if tsOri, err = f.db.Get(c, key); err != nil {
			return false
		}

		if ts, err := strconv.ParseInt(tsOri, 10, 64); err != nil || ts > times {
			return false
		}
		return true
	})
	return
}

// Incr increments the count only.
func (f *Freq) Incr(c context.Context, pre string, rule ...FreqRule) (bRst bool, err error) {
	bRst = f.freq(pre, rule, func(key string, expire, times int64) bool {
		var tsOri interface{}
		if tsOri, err = f.db.Eval(c, incrLuaScript, []string{key}, []interface{}{expire}); err != nil {
			return false
		}

		if ts, ok := tsOri.(int64); !ok || ts == -1 {
			return false
		}
		return true
	})
	return
}

// IncrCheck increments and checks the count.
func (f *Freq) IncrCheck(c context.Context, pre string, rule ...FreqRule) (bRst bool, err error) {
	bRst = f.freq(pre, rule, func(key string, expire, times int64) bool {
		var tsOri interface{}
		if tsOri, err = f.db.Eval(c, incrLuaScript, []string{key}, []interface{}{expire}); err != nil {
			return false
		}

		if ts, ok := tsOri.(int64); !ok || ts == -1 || ts > times {
			return false
		}
		return true
	})
	return
}

func (f *Freq) freq(pre string, ruleList []FreqRule, fn func(key string, expire, times int64) bool) bool {
	prekey := lib.StrJoin("freq:", pre, ":")
	for _, r := range ruleList {
		var key string
		var expire int64
		switch r.Duri {
		case "today":
			now := time.Now()
			tomorrowFirst := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, f.tz)
			key = prekey + now.Format("2006_01_02")
			expire = int64(tomorrowFirst.Sub(now).Seconds())
		default:
			var err error
			key = prekey + r.Duri
			expire, err = strconv.ParseInt(r.Duri, 10, 64)
			if nil != err {
				return false
			}
		}
		if false == fn(key, expire, r.Times) {
			return false
		}
	}
	return true
}
