package inout

/*
 * Verification,conversion and filter for paramter
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-03
 * @demo1 NewVCF(map[string]IDo{
		 "int641": inout.NewInt().IsGte(10).IsLte(90).InInt64(20),
		 "str1": inout.NewStr("deff").IsGte(4).IsGte(10).InStr("asdfghjk"),
	 }).Do()
 * @demo2 NewVCF(map[string]IDo{
		 "int641": inout.NewInt().IsGte(10).IsLte(90),
		 "str1": inout.NewStr("deff").IsGte(4).IsGte(10),
	 }).InValueByStruct(&a{Num: 80, Str: "bbbbb"}).Do()
 * @demo3 NewVCF(map[string]IDo{
		 "int641": inout.NewInt().IsGte(10).IsLte(90),
		 "str1": inout.NewStr("deff").IsGte(4).IsLte(10),
	 }).InValueByStrMap(&map[string]string{"Num": "80", "Str": "bbbbb"}).Do()
*/

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/neo532/gofr/lib"
)

//V* Verification
//C* Conversion
//F* Filter
var (
	Venum      = regexp.MustCompile(`^[\w_,]{0,200}$`)
	Vint       = regexp.MustCompile(`^\d{0,18}$`)
	Vnum       = regexp.MustCompile(`^[-\d.]{0,50}$`)
	Vversion   = regexp.MustCompile(`^\d(.\d+)*$`)
	VmobileCn  = regexp.MustCompile(`^1[^012]\d{9}$`)
	VnoSpecial = regexp.MustCompile(`^[^'";$` + "`" + `]*$`)
	Vbase64    = regexp.MustCompile(`^(?:[A-Za-z0-99+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$`)
	Vemail     = regexp.MustCompile(`^[\w!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[\w!#$%&'*+/=?^_` + "`" + `{|}~-]+)*@(?:[\w](?:[\w-]*[\w])?\.)+[a-zA-Z0-9](?:[\w-]*[\w])?$`)
)

// VerificationConversionFilter is a instance for Verification and Conversion.
// It contains the error and the list of function
type VerificationConversionFilter struct {
	err    error
	fnList map[string]IDo
}

// NewVCF returns a instance of VerificationConversionFilter by map of IDo.
func NewVCF(doList map[string]IDo) *VerificationConversionFilter {
	return &VerificationConversionFilter{
		fnList: doList,
	}
}

// Do executes the this VerificationConversionFilter.
func (vc *VerificationConversionFilter) Do() *VerificationConversionFilter {
	for field, doer := range vc.fnList {
		if err := doer.Do(); err != "" {
			vc.err = errors.New(lib.StrJoin(field, ":", err))
			return vc
		}
	}
	return vc
}

// InValueByStrMap inputs one map of string into this VerificationConversionFilter.
func (vc *VerificationConversionFilter) InValueByStrMap(mapDL map[string]string) *VerificationConversionFilter {
	for k, v := range mapDL {
		vc.fnList[k].InStr(v)
	}
	return vc
}

// InValueByStruct inputs one struct into this VerificationConversionFilter.
func (vc *VerificationConversionFilter) InValueByStruct(obj interface{}) *VerificationConversionFilter {
	objT := reflect.TypeOf(obj)
	objV := reflect.ValueOf(obj)
	switch {
	case objT.Kind() == reflect.Struct:
	case objT.Kind() == reflect.Ptr && objT.Elem().Kind() == reflect.Struct:
		objT = objT.Elem()
		objV = objV.Elem()
	default:
		vc.err = fmt.Errorf("%v must be a struct or a struct pointer", obj)
		return vc
	}
	for i := 0; i < objT.NumField(); i++ {
		field := objT.Field(i)
		fieldName := strings.ToLower(field.Name)
		switch objV.Field(i).Kind() {
		case reflect.String:
			vc.fnList[fieldName].InValue(objV.FieldByName(field.Name).String())
		case reflect.Float64:
			vc.fnList[fieldName].InValue(objV.FieldByName(field.Name).Float())
		case reflect.Int64:
		case reflect.Int:
			vc.fnList[fieldName].InValue(objV.FieldByName(field.Name).Int())
		default:
			vc.err = fmt.Errorf(
				"%v isnot support type. string/float64/int64/int only",
				objV.Field(i).Kind(),
			)
			return vc
		}
	}
	return vc
}

// IsOk returns the total result of this VerificationConversionFilter by boolean.
func (vc *VerificationConversionFilter) IsOk() bool {
	return nil == vc.err
}

// Err returns the error of this VerificationConversionFilter by error.
func (vc *VerificationConversionFilter) Err() error {
	return vc.err
}

// Int returns this paramter result by int.
func (vc *VerificationConversionFilter) Int(field string) int {
	return vc.fnList[field].Value().(int)
}

// Int64 returns this paramter result by int64.
func (vc *VerificationConversionFilter) Int64(field string) int64 {
	return vc.fnList[field].Value().(int64)
}

// Float64 returns this paramter result by float64.
func (vc *VerificationConversionFilter) Float64(field string) float64 {
	return vc.fnList[field].Value().(float64)
}

// String returns this paramter result by String.
func (vc *VerificationConversionFilter) String(field string) string {
	return vc.fnList[field].Value().(string)
}

//========== rule ==========

// IDo interface is to be implemented by Verification and Conversion.
type IDo interface {
	Do() string
	Value() interface{}
	InStr(str string) IDo
	InValue(val interface{}) IDo
}

//---------- Int ----------

// Int is a instance of IDo for parameter of int.
type Int struct {
	gte   int64
	lte   int64
	inArr []int

	inValue int64
	err     string
	def     int64
	value   int64
	fnList  []func() string
}

// NewInt returns a instance of Int.
// You can set the default when VerificationConversionFilter has error.
func NewInt(d ...int64) *Int {
	if len(d) == 1 {
		return &Int{
			def: d[0],
		}
	}
	return &Int{}
}

// InStr Inputs a paramter by String.
func (i *Int) InStr(v string) IDo {
	var err error
	if i.inValue, err = strconv.ParseInt(v, 10, 64); nil != err {
		i.err = err.Error()
	}
	return i
}

// InInt64 Inputs a paramter by int64.
func (i *Int) InInt64(v int64) IDo {
	i.inValue = v
	return i
}

// InInt Inputs a paramter by int.
func (i *Int) InInt(v int) IDo {
	i.inValue = int64(v)
	return i
}

// InValue Inputs a paramter by interface{}.
func (i *Int) InValue(v interface{}) IDo {
	i.inValue = v.(int64)
	return i
}

// IsGte verifys a value whether it is great and equal than the input.
func (i *Int) IsGte(gte int) *Int {
	i.gte = int64(gte)
	i.fnList = append(i.fnList, func() string {
		if i.inValue < i.gte {
			return "Value is too small!"
		}
		return ""
	})
	return i
}

// IsLte verifys a value whether it is less and equal than the input.
func (i *Int) IsLte(lte int) *Int {
	i.lte = int64(lte)
	i.fnList = append(i.fnList, func() string {
		if i.inValue > i.lte {
			return "Value is too large!"
		}
		return ""
	})
	return i
}

// IsInArr verifys a value whether it is in this array.
func (i *Int) IsInArr(enumList ...int) *Int {
	i.inArr = enumList
	i.fnList = append(i.fnList, func() string {
		for _, v := range i.inArr {
			if v == int(i.inValue) {
				return ""
			}
		}
		return "Don't have i item."
	})
	return i
}

// Value returns this value by interface{}.
func (i *Int) Value() interface{} {
	return i.value
}

// Do excute this type of VCF and return the message of error.
func (i *Int) Do() string {
	for _, fn := range i.fnList {
		if i.err != "" {
			return i.err
		}
		if err := fn(); err != "" {
			if 0 != i.def {
				i.value = i.def
			}
			return err
		}
		i.value = i.inValue
	}
	return ""
}

//---------- /Int ----------
//---------- Float ----------

// Float is a instance of IDo for parameter of float.
type Float struct {
	gte float64
	lte float64

	inValue float64
	err     string
	def     float64
	value   float64
	fnList  []func() string
}

// NewFloat returns a instance of Float.
// You can set the default when VerificationConversionFilter has error.
func NewFloat(d ...float64) *Float {
	if len(d) == 1 {
		return &Float{
			def: d[0],
		}
	}
	return &Float{}
}

// InStr Inputs a paramter by string.
func (f *Float) InStr(v string) IDo {
	var err error
	if f.inValue, err = strconv.ParseFloat(v, 64); nil != err {
		f.err = err.Error()
	}
	return f
}

// InFloat64 Inputs a paramter by float64.
func (f *Float) InFloat64(v float64) IDo {
	f.inValue = v
	return f
}

// InValue Inputs a paramter by interface{}.
func (f *Float) InValue(v interface{}) IDo {
	f.inValue = v.(float64)
	return f
}

// IsGte verifys a value whether it is great and equal than the input.
func (f *Float) IsGte(gte float64) *Float {
	f.gte = gte
	f.fnList = append(f.fnList, func() string {
		if f.inValue < f.gte {
			return "Value is too small!"
		}
		return ""
	})
	return f
}

// IsLte verifys a value whether it is less and equal than the input.
func (f *Float) IsLte(lte float64) *Float {
	f.lte = lte
	f.fnList = append(f.fnList, func() string {
		if f.inValue > f.lte {
			return "Value is too large!"
		}
		return ""
	})
	return f
}

// Value returns this value by interface{}.
func (f *Float) Value() interface{} {
	return f.value
}

// Do excute this type of VCF and return the message of error.
func (f *Float) Do() string {
	for _, fn := range f.fnList {
		if f.err != "" {
			return f.err
		}
		if err := fn(); err != "" {
			if 0 != f.def {
				f.value = f.def
			}
			return err
		}
		f.value = f.inValue
	}
	return ""
}

//---------- /Int ----------
//---------- String ----------

// String is a instance of IDo for parameter of string.
type String struct {
	gte    int
	lte    int
	regexp *regexp.Regexp
	inArr  []string
	inMap  map[string]string

	inValue string
	err     string
	def     string
	value   string
	fnList  []func() string
}

// NewStr returns a instance of String.
// You can set the default when VerificationConversionFilter has error.
func NewStr(d ...string) *String {
	if len(d) == 1 {
		return &String{
			def: d[0],
		}
	}
	return &String{}
}

// InStr Inputs a paramter by string.
func (s *String) InStr(v string) IDo {
	s.inValue = v
	return s
}

// InValue Inputs a paramter by interface{}.
func (s *String) InValue(v interface{}) IDo {
	s.inValue = v.(string)
	return s
}

// IsGte verifys a value's length whether it is great and equal than the input.
func (s *String) IsGte(gte int) *String {
	s.gte = gte
	s.fnList = append(s.fnList, func() string {
		if len(s.inValue) < s.gte {
			return "Length is too short!"
		}
		return ""
	})
	return s
}

// IsLte verifys a value' length whether it is less and equal than the input.
func (s *String) IsLte(lte int) *String {
	s.lte = lte
	s.fnList = append(s.fnList, func() string {
		if len(s.inValue) > s.lte {
			return "Length is too long!"
		}
		return ""
	})
	return s
}

// RegExp verifys a value whether it matches the regular expression.
func (s *String) RegExp(exp *regexp.Regexp) *String {
	s.regexp = exp
	s.fnList = append(s.fnList, func() string {
		if exp.MatchString(s.inValue) {
			return ""
		}
		return "Wrong rule."
	})
	return s
}

// IsInMap verifys a value whether the map contain this value.
func (s *String) IsInMap(mapList map[string]string) *String {
	s.inMap = mapList
	s.fnList = append(s.fnList, func() string {
		if _, ok := s.inMap[s.inValue]; ok {
			return ""
		}
		return "Don't have s item."
	})
	return s
}

// Slash converts the value by quote.
func (s *String) Slash() *String {
	s.fnList = append(s.fnList, func() string {
		return strconv.Quote(s.inValue)
	})
	return s
}

// IsInArr verifys a value whether it is in this array.
func (s *String) IsInArr(enumList ...string) *String {
	s.inArr = enumList
	s.fnList = append(s.fnList, func() string {
		for _, v := range s.inArr {
			if v == s.inValue {
				return ""
			}
		}
		return "Don't have s item."
	})
	return s
}

// Value returns this value by interface{}.
func (s *String) Value() interface{} {
	return s.value
}

// Do excute this type of VCF and return the message of error.
func (s *String) Do() string {
	for _, fn := range s.fnList {
		if s.err != "" {
			return s.err
		}
		if err := fn(); err != "" {
			if "" != s.def {
				s.value = s.def
			}
			return err
		}
		s.value = s.inValue
	}
	return ""
}

//---------- /String ----------
//========== /rule ==========
