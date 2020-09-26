/*
 * @abstract crypt
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package lib

import (
	"math/big"
	"strings"
)

const (
	DICT36 = "abcdefghijklmnopqrstuvwxyz1234567890"
)

func EnNum2strByDict(num int64, dict string) string {
	var str strings.Builder
	lenD := int64(len(dict))
	for {
		if num <= 0 {
			break
		}
		str.WriteString(string(dict[num%lenD]))
		num = num / lenD
	}
	return Reverse(str.String())
}

func DeStr2numByDict(str string, dict string) int64 {
	lenD := len(dict)
	lenS := len(str)

	var rst = big.NewInt(0)
	var j = big.NewInt(1)
	var pos int
	for i := 0; i < lenS; i++ {
		pos = strings.Index(dict, string(str[i]))

		j = j.Exp(big.NewInt(int64(lenD)), big.NewInt(int64(lenS-i-1)), nil)
		j = j.Mul(big.NewInt(int64(pos)), j)
		rst = rst.Add(rst, j)
	}
	return rst.Int64()
}
