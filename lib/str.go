/*
 * @abstract string
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package lib

import (
	"math"
	"strconv"
	"strings"
)

// Reverse returns the string after reversing.
func Reverse(str string) string {
	runes := []rune(str)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Ucfirst returns the string,the first letter is upper.
func Ucfirst(str string) string {
	if len(str) < 1 {
		return ""
	}
	strArry := []rune(str)
	if strArry[0] >= 97 && strArry[0] <= 122 {
		strArry[0] -= 32
	}
	return string(strArry)
}

// StrJoin joins strings by the array of string.
func StrJoin(args ...string) string {
	var b strings.Builder
	for _, str := range args {
		b.WriteString(str)
	}
	return b.String()
}

// StrBJoin joins strings by a builder and the array of string.
func StrBJoin(b *strings.Builder, args ...string) {
	for _, str := range args {
		b.WriteString(str)
	}
}

// Float2str returns the string after converting the float64 to string.
func Float2str(num float64, decimal int) string {
	d := float64(1)
	if decimal > 0 {
		d = math.Pow10(decimal)
	}
	return strconv.FormatFloat(math.Trunc(num*d)/d, 'f', -1, 64)
}
