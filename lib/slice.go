package lib

import "sort"

/*
 * @abstract string
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-02
 */

// IArray
type IArray interface {
	Len() int
	Value(i int) interface{}
	Append(v interface{}) IArray
	Make() IArray
	LessValue(i int, v interface{}) bool

	Less(i, j int) bool
	Swap(i, j int)
	IsRighType(v interface{}) bool
}

// UniqArray
func UniqArray(ori IArray) IArray {
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

// MinusArray
func MinusArray(m, s IArray) (rst IArray) {
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

// IntersectArray
func IntersectArray(o, t IArray) (rst IArray) {
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

// InSortArray
func InSortArray(needle interface{}, haystack IArray) bool {
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

// Uint64Array
type Uint64Array []uint64

func (a Uint64Array) Len() int                            { return len(a) }
func (a Uint64Array) Value(i int) interface{}             { return a[i] }
func (a Uint64Array) Append(v interface{}) IArray         { return append(a, v.(uint64)) }
func (a Uint64Array) Make() IArray                        { return make(Uint64Array, 0, a.Len()) }
func (a Uint64Array) Less(i, j int) bool                  { return a.Value(i).(uint64) > a.Value(j).(uint64) }
func (a Uint64Array) LessValue(i int, v interface{}) bool { return a.Value(i).(uint64) > v.(uint64) }
func (a Uint64Array) Swap(i, j int)                       { a[i], a[j] = a[j], a[i] }
func (a Uint64Array) IsRighType(v interface{}) bool {
	switch v.(type) {
	case uint64:
		return true
	default:
		return false
	}
}

// StringArray
type StringArray []string

func (a StringArray) Len() int                            { return len(a) }
func (a StringArray) Value(i int) interface{}             { return a[i] }
func (a StringArray) Append(v interface{}) IArray         { return append(a, v.(string)) }
func (a StringArray) Make() IArray                        { return make(StringArray, 0, a.Len()) }
func (a StringArray) Less(i, j int) bool                  { return a.Value(i).(string) > a.Value(j).(string) }
func (a StringArray) LessValue(i int, v interface{}) bool { return a.Value(i).(string) > v.(string) }
func (a StringArray) Swap(i, j int)                       { a[i], a[j] = a[j], a[i] }
func (a StringArray) IsRighType(v interface{}) bool {
	switch v.(type) {
	case string:
		return true
	default:
		return false
	}
}
