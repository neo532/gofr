package slices

import (
	"math/rand"
	"sort"
)

/*
 * @abstract ISlice
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-02
 */

// ISlice is a interface for multi-type.
type ISlice interface {
	Len() int
	Value(i int) interface{}
	Append(v interface{}) ISlice
	Make() ISlice
	LessValue(i int, v interface{}) bool

	Less(i, j int) bool
	Swap(i, j int)
	IsRighType(v interface{}) bool
}

// Unique handles ISlice.
func Unique(ori ISlice) ISlice {
	var isUniq bool
	var lenRst int
	var lenOri = ori.Len()
	var rst = ori.Make()
	for i := 0; i < lenOri; i++ {
		lenRst = rst.Len()
		isUniq = true
		for j := 0; j < lenRst; j++ {
			if ori.Value(i) == rst.Value(j) {
				isUniq = false
				break
			}
		}
		if isUniq == true {
			rst = rst.Append(ori.Value(i))
		}
	}
	return rst
}

// Minus handles ISlice.
func Minus(m, s ISlice) (rst ISlice) {
	var isHas bool
	var lenM = m.Len()
	var lenS = s.Len()
	rst = m.Make()
	for i := 0; i < lenM; i++ {
		isHas = false
		for j := 0; j < lenS; j++ {
			if m.Value(i) == s.Value(j) {
				isHas = true
				break
			}
		}
		if isHas == false {
			rst = rst.Append(m.Value(i))
		}
	}
	return
}

// Intersect handles ISlice.
func Intersect(o, t ISlice) (rst ISlice) {
	var lenO = o.Len()
	var lenT = t.Len()
	rst = o.Make()
	if lenO == 0 || lenT == 0 {
		return
	}
	for i := 0; i < lenO; i++ {
		for j := 0; j < lenT; j++ {
			if o.Value(i) == t.Value(j) {
				rst = rst.Append(o.Value(i))
				break
			}
		}
	}
	return rst
}

// In handles ISlice.
func In(needle interface{}, haystack ISlice) bool {
	var l = haystack.Len()
	if l == 0 || haystack.IsRighType(needle) == false {
		return false
	}
	for i := 0; i < l; i++ {
		if needle == haystack.Value(i) {
			return true
		}
	}
	return false
}

// InSort handles ISlice.
func InSort(needle interface{}, haystack ISlice) bool {
	var l = haystack.Len()
	if l == 0 || haystack.IsRighType(needle) == false {
		return false
	}
	var index = sort.Search(l, func(i int) bool {
		return haystack.LessValue(i, needle)
	})
	if index < l && haystack.Value(index) == needle {
		return true
	}
	return false
}

// Shuffle shuffles the data's order.
func Shuffle(data ISlice) {
	for i := data.Len() - 1; i > 0; i-- {
		data.Swap(i, rand.Intn(i+1))
	}
}
