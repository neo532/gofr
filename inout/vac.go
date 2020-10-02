package inout

/*
 * Verification and Conversion for paramter
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 * @demo NewVaC(map[string]IDo{
	 "int641": inout.NewInt().IsGte(10).IsLte(90).InInt64(20),
	 "str1": inout.NewStr("deff").IsGte(4).IsGte(10).InStr("asdfghjk"),
 }).Do()
*/

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/neo532/gofr/lib"
)

const (
	C_ENUM       = `^[\w_,]{0,200}$`
	C_INT        = `^\d{0,18}$`
	C_NUM        = `^[-\d.]{0,50}$`
	C_VERION     = `^\d(.\d+)*$`
	C_MOBILE_CN  = `^1[^012]\d{9}$`
	C_NO_SPECIAL = `^[^'";$\\]*$`
)

type verificationConversion struct {
	err    error
	fnList map[string]IDo
}

func NewVaC(doList map[string]IDo) *verificationConversion {
	return &verificationConversion{
		fnList: doList,
	}
}
func (this *verificationConversion) Do() *verificationConversion {
	for field, doer := range this.fnList {
		if err := doer.Do(); err != "" {
			this.err = errors.New(lib.StrJoin(field, ":", err))
			return this
		}
	}
	return this
}

//func (this *verificationConversion) InValue(obj interface{}) *verificationConversion {
//	objT := reflect.TypeOf(obj)
//	objV := reflect.ValueOf(obj)
//	switch {
//	case objT.Kind() == reflect.Struct:
//	case objT.Kind() == reflect.Ptr && objT.Elem().Kind() == reflect.Struct:
//		objT = objT.Elem()
//		objV = objV.Elem()
//	default:
//		this.err = fmt.Errorf("%v must be a struct or a struct pointer", obj)
//		return this
//	}
//	for i := 0; i < objT.NumField(); i++ {
//		field := objT.Field(i)
//		fieldName := strings.ToLower(field.Name)
//		switch objV.Field(i).Kind() {
//		case reflect.String:
//			this.fnList[fieldName].InStr(objV.FieldByName(field.Name).String())
//		case reflect.Int64:
//			this.fnList[fieldName].InInt64(objV.FieldByName(field.Name).String())
//		case reflect.Int:
//			this.fnList[fieldName].InInt(objV.FieldByName(field.Name).String())
//		}
//	}
//	return this
//}
func (this *verificationConversion) IsOk() bool {
	return nil == this.err
}
func (this *verificationConversion) Int64(field string) int64 {
	return this.fnList[field].Value().(int64)
}
func (this *verificationConversion) String(field string) string {
	return this.fnList[field].Value().(string)
}
func (this *verificationConversion) Err() error {
	return this.err
}

//========== rule ==========

type IDo interface {
	Do() string
	Value() interface{}
	InStr(str string) IDo
	InValue(val interface{}) IDo
}

type Int struct {
	gte     int64
	lte     int64
	inArr   []int
	inValue int64

	err    string
	def    int64
	value  int64
	fnList []func() string
}

func NewInt(d ...int64) *Int {
	if len(d) == 1 {
		return &Int{
			def: d[0],
		}
	}
	return &Int{}
}
func (this *Int) InStr(v string) IDo {
	var err error
	if this.inValue, err = strconv.ParseInt(v, 10, 64); nil != err {
		this.err = err.Error()
	}
	return this
}
func (this *Int) InInt64(v int64) IDo {
	this.inValue = v
	return this
}
func (this *Int) InInt(v int) IDo {
	this.inValue = int64(v)
	return this
}
func (this *Int) InValue(v interface{}) IDo {
	this.inValue = v.(int64)
	return this
}
func (this *Int) IsGte(gte int) *Int {
	this.gte = int64(gte)
	this.fnList = append(this.fnList, func() string {
		if this.inValue < this.gte {
			return "Value is too small!"
		}
		return ""
	})
	return this
}
func (this *Int) IsLte(lte int) *Int {
	this.lte = int64(lte)
	this.fnList = append(this.fnList, func() string {
		if this.inValue > this.lte {
			return "Value is too large!"
		}
		return ""
	})
	return this
}
func (this *Int) IsInArr(enumList ...int) *Int {
	this.inArr = enumList
	this.fnList = append(this.fnList, func() string {
		for _, v := range this.inArr {
			if v == int(this.inValue) {
				return ""
			}
		}
		return "Don't have this item."
	})
	return this
}
func (this *Int) Value() interface{} {
	return this.value
}
func (this *Int) Do() string {
	for _, fn := range this.fnList {
		if this.err != "" {
			return this.err
		}
		if err := fn(); err != "" {
			if 0 != this.def {
				this.value = this.def
			}
			return err
		}
		this.value = this.inValue
	}
	return ""
}

type String struct {
	gte     int
	lte     int
	regexp  string
	inArr   []string
	inMap   map[string]string
	inValue string

	err    string
	def    string
	value  string
	fnList []func() string
}

func NewStr(d ...string) *String {
	if len(d) == 1 {
		return &String{
			def: d[0],
		}
	}
	return &String{}
}
func (this *String) InStr(v string) IDo {
	this.inValue = v
	return this
}
func (this *String) InValue(v interface{}) IDo {
	this.inValue = v.(string)
	return this
}
func (this *String) IsGte(gte int) *String {
	this.gte = gte
	this.fnList = append(this.fnList, func() string {
		if len(this.inValue) < this.gte {
			return "Length is too short!"
		}
		return ""
	})
	return this
}
func (this *String) IsLte(lte int) *String {
	this.lte = lte
	this.fnList = append(this.fnList, func() string {
		if len(this.inValue) > this.gte {
			return "Length is too long!"
		}
		return ""
	})
	return this
}
func (this *String) RegExp(exp string) *String {
	this.regexp = exp
	this.fnList = append(this.fnList, func() string {
		if ok, _ := regexp.MatchString(this.regexp, this.inValue); ok {
			return ""
		}
		return "Wrong rule."
	})
	return this
}
func (this *String) IsInMap(mapList map[string]string) *String {
	this.inMap = mapList
	this.fnList = append(this.fnList, func() string {
		if _, ok := this.inMap[this.inValue]; ok {
			return ""
		}
		return "Don't have this item."
	})
	return this
}

func (this *String) Slash() *String {
	this.fnList = append(this.fnList, func() string {
		return strconv.Quote(this.inValue)
	})
	return this
}
func (this *String) IsInArr(enumList ...string) *String {
	this.inArr = enumList
	this.fnList = append(this.fnList, func() string {
		for _, v := range this.inArr {
			if v == this.inValue {
				return ""
			}
		}
		return "Don't have this item."
	})
	return this
}
func (this *String) Value() interface{} {
	return this.value
}
func (this *String) Do() string {
	for _, fn := range this.fnList {
		if this.err != "" {
			return this.err
		}
		if err := fn(); err != "" {
			if "" != this.def {
				this.value = this.def
			}
			return err
		}
		this.value = this.inValue
	}
	return ""
}

//========== /rule ==========
