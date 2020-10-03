/*
 * @abstract tool
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package lib

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
		EnNum2strByDict(int64(Ip2long(LocalIp())), DICT36),
		EnNum2strByDict(int64(os.Getpid()), DICT36),
	)
}

func SetParam(p ...string) {
	if len(p) > 0 {
		ip = p[0]
	}
	if len(p) > 1 {
		pid = p[1]
	}
	param = ip + pid
}

func UniqId(pre string) string {
	return StrJoin(
		pre,
		param,
		strconv.FormatInt(time.Now().Unix(), 10),
		strconv.FormatUint(counter.Get(), 10),
	)
}
