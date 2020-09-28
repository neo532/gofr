/*
 * @abstract frequency control
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package tool

import (
	"strconv"
	"time"

	"github.com/neo532/gofr/lib"
)

type IFreqDb interface {
	Incr(key string) (int64, error)
	Expire(key string, expiration time.Duration) (bool, error)
	Get(key string) (string, error)
}

type FreqRule struct {
	Duri  string //3|day
	Times int
}

type Freq struct {
	tz *time.Location
	db IFreqDb
}

func NewFreq(d IFreqDb) *Freq {
	return &Freq{
		db: d,
	}
}

func (this *Freq) Timezone(timezone string) *Freq {
	this.tz, _ = time.LoadLocation(timezone)
	return this
}

func (this *Freq) Incr(pre string, rule ...FreqRule) bool {
	return this.freq(pre, rule, func(key string, expire, times int) bool {
		tsOri, err := this.db.Incr(key)
		if nil != err {
			return false
		}
		ts := int(tsOri)
		if ts == 1 {
			this.db.Expire(key, time.Duration(expire)*time.Second)
		}
		return true
	})
}

func (this *Freq) Check(pre string, rule ...FreqRule) bool {
	return this.freq(pre, rule, func(key string, expire, times int) bool {
		tsOri, e1 := this.db.Get(key)
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

func (this *Freq) IncrCheck(pre string, rule ...FreqRule) bool {
	return this.freq(pre, rule, func(key string, expire, times int) bool {
		tsOri, e1 := this.db.Incr(key)
		if nil != e1 {
			return false
		}
		ts := int(tsOri)
		if ts == 1 {
			this.db.Expire(key, time.Duration(int64(expire))*time.Second)
		}
		if ts > times {
			return false
		}
		return true
	})
}

func (this *Freq) freq(pre string, ruleList []FreqRule, fn func(key string, expire, times int) bool) bool {
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
