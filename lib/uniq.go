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

func SetParam(param ...string) {
	if len(param) > 0 {
		ip = param[0]
	}
	if len(param) > 1 {
		pid = param[1]
	}
	param = ip + pid
}

func UniqId(pre string) string {
	return StrJoin(
		param,
		strconv.FormatInt(time.Now().Unix(), 10),
		strconv.FormatUint(counter.Get(), 10),
	)
}
