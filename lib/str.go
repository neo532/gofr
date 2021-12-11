package lib

/*
 * @abstract string
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"bytes"
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
	var b bytes.Buffer
	for _, str := range args {
		b.WriteString(str)
	}
	return b.String()
}

// StrBJoin joins strings by a builder and the array of string.
func StrBJoin(b bytes.Buffer, args ...string) bytes.Buffer {
	for _, str := range args {
		b.WriteString(str)
	}
	return b
}

// Float2str returns the string after converting the float64 to string.
func Float2str(num float64, decimal int) string {
	d := float64(1)
	if decimal > 0 {
		d = math.Pow10(decimal)
	}
	return strconv.FormatFloat(math.Trunc(num*d)/d, 'f', -1, 64)
}

// String is a type of converter for string.
type String string

// Int returns the int after converting the string to int.
func (s String) Int() int {
	if i, err := strconv.Atoi(strings.Split(string(s), ".")[0]); nil == err {
		return i
	}
	return 0
}

// Int64 returns the int64 after converting the string to int64.
func (s String) Int64() int64 {
	if i, err := strconv.ParseInt(strings.Split(string(s), ".")[0], 10, 64); nil == err {
		return i
	}
	return 0
}

// Float64 returns the float64 after converting the string to float64.
func (s String) Float64() float64 {
	if i, err := strconv.ParseFloat(string(s), 64); nil == err {
		return i
	}
	return 0
}
