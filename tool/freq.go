package tool

/*
 * @abstract frequency control
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"strconv"
	"time"

	"github.com/neo532/gofr/lib"
)

// IFreqDb is the interface for FreqRule.
type IFreqDb interface {
	Incr(key string) (int64, error)
	Expire(key string, expiration time.Duration) (bool, error)
	Get(key string) (string, error)
}

// FreqRule is the instance for FreqRule.
type FreqRule struct {
	Duri  string //3|day
	Times int
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
func (f *Freq) Timezone(timezone string) *Freq {
	f.tz, _ = time.LoadLocation(timezone)
	return f
}

// Incr increments the count only.
func (f *Freq) Incr(pre string, rule ...FreqRule) bool {
	return f.freq(pre, rule, func(key string, expire, times int) bool {
		tsOri, err := f.db.Incr(key)
		if nil != err {
			return false
		}
		ts := int(tsOri)
		if ts == 1 {
			f.db.Expire(key, time.Duration(expire)*time.Second)
		}
		return true
	})
}

// Check checks the count only.
func (f *Freq) Check(pre string, rule ...FreqRule) bool {
	return f.freq(pre, rule, func(key string, expire, times int) bool {
		tsOri, e1 := f.db.Get(key)
		if nil != e1 {
			return false
		}
		ts, e2 := strconv.Atoi(tsOri)
		if nil != e2 || ts > times {
			return false
		}
		return true
	})
}

// IncrCheck increments and checks the count.
func (f *Freq) IncrCheck(pre string, rule ...FreqRule) bool {
	return f.freq(pre, rule, func(key string, expire, times int) bool {
		tsOri, e1 := f.db.Incr(key)
		if nil != e1 {
			return false
		}
		ts := int(tsOri)
		if ts == 1 {
			f.db.Expire(key, time.Duration(int64(expire))*time.Second)
		}
		if ts > times {
			return false
		}
		return true
	})
}

func (f *Freq) freq(pre string, ruleList []FreqRule, fn func(key string, expire, times int) bool) bool {
	var prekey = lib.StrJoin("freq:", pre, ":")
	for _, r := range ruleList {
		var key string
		var expire int
		switch r.Duri {
		case "day":
			key = prekey + time.Now().Format("2006_01_02")
			expire = 86400
		default:
			var err error
			key = prekey + r.Duri
			expire, err = strconv.Atoi(r.Duri)
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
