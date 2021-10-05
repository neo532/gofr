package lib

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

// ISlice
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

// UniqSlice
func UniqSlice(ori ISlice) ISlice {
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

// MinusSlice
func MinusSlice(m, s ISlice) (rst ISlice) {
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

// IntersectSlice
func IntersectSlice(o, t ISlice) (rst ISlice) {
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

// InSortSlice
func InSortSlice(needle interface{}, haystack ISlice) bool {
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

// ShuffleSlice shuffles the data's order.
func ShuffleSlice(data ISlice) {
	for i := data.Len() - 1; i > 0; i-- {
		data.Swap(i, rand.Intn(i+1))
	}
}

// Uint64Slice
type Uint64Slice []uint64

func (a Uint64Slice) Len() int                            { return len(a) }
func (a Uint64Slice) Value(i int) interface{}             { return a[i] }
func (a Uint64Slice) Append(v interface{}) ISlice         { return append(a, v.(uint64)) }
func (a Uint64Slice) Make() ISlice                        { return make(Uint64Slice, 0, a.Len()) }
func (a Uint64Slice) Less(i, j int) bool                  { return a.Value(i).(uint64) > a.Value(j).(uint64) }
func (a Uint64Slice) LessValue(i int, v interface{}) bool { return a.Value(i).(uint64) > v.(uint64) }
func (a Uint64Slice) Swap(i, j int)                       { a[i], a[j] = a[j], a[i] }
func (a Uint64Slice) IsRighType(v interface{}) bool {
	switch v.(type) {
	case uint64:
		return true
	default:
		return false
	}
}

// StringSlice
type StringSlice []string

func (a StringSlice) Len() int                            { return len(a) }
func (a StringSlice) Value(i int) interface{}             { return a[i] }
func (a StringSlice) Append(v interface{}) ISlice         { return append(a, v.(string)) }
func (a StringSlice) Make() ISlice                        { return make(StringSlice, 0, a.Len()) }
func (a StringSlice) Less(i, j int) bool                  { return a.Value(i).(string) > a.Value(j).(string) }
func (a StringSlice) LessValue(i int, v interface{}) bool { return a.Value(i).(string) > v.(string) }
func (a StringSlice) Swap(i, j int)                       { a[i], a[j] = a[j], a[i] }
func (a StringSlice) IsRighType(v interface{}) bool {
	switch v.(type) {
	case string:
		return true
	default:
		return false
	}
}
