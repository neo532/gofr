package lib

/*
 * @abstract tool
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"strconv"
	"strings"
)

const (
	Larger  = 1
	Smaller = -1
	Equal   = 0
	Error   = -2
)

// CompareVersion returns the num after comparing two versions.
//  1: ver1 > ver2
//  0: ver1 = ver2
// -1: ver1 < ver2
// -2: has error
func CompareVersion(ver1, ver2 string) int {
	larger := Larger
	smaller := Smaller
	v1 := strings.Split(strings.Trim(ver1, "."), ".")
	v2 := strings.Split(strings.Trim(ver2, "."), ".")
	v1Len := len(v1)
	v2Len := len(v2)

	//make sure that v1's length is longer than v2.
	if v1Len < v2Len {
		v1, v2 = v2, v1
		v1Len, v2Len = v2Len, v1Len
		larger, smaller = smaller, larger
	}

	var v1i, v2i int
	var err error
	for i := 0; i < v2Len; i++ {
		if v1i, err = strconv.Atoi(v1[i]); err != nil {
			return Error
		}
		if v2i, err = strconv.Atoi(v2[i]); err != nil {
			return Error
		}
		if v1i > v2i {
			return larger
		}
		if v1i < v2i {
			return smaller
		}
	}

	if v1Len != v2Len {
		return larger
	}
	return Equal
}
