package lib

/*
 * @abstract tool
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"os"
	"strconv"
	"time"
)

var (
	param   string
	counter *AutoInc
	ip      string
	pid     string
)

func init() {
	counter = NewAutoInc(0, 1)
	SetParam(
		Num2strByDict(int64(IP2long(LocalIP())), DICT36),
		Num2strByDict(int64(os.Getpid()), DICT36),
	)
}

// SetParam sets prefix for unique string.
func SetParam(p ...string) {
	if len(p) > 0 {
		ip = p[0]
	}
	if len(p) > 1 {
		pid = p[1]
	}
	param = ip + pid + Num2strByDict(time.Now().Unix(), DICT36)
}

// UniqID returns a unique string.
func UniqID(pre string) string {
	return StrJoin(
		pre,
		param,
		strconv.FormatUint(counter.Get(), 10),
	)
}
