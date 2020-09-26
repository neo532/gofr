package inout

/*
 * Verification and Conversion for paramter
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/neo532/gofr/lib"
)

type verificationConversion struct {
	inParam  map[string]string
	outParam map[string]string
	err      string
}

const (
	ENUM       = `^[\w_,]{0,200}$`
	INT        = `^\d{0,20}$`
	NUM        = `^[-\d.]{0,50}$`
	VERION     = `^\d(.\d+)*$`
	MOBILE_CN  = `^1[^012]\d{9}$`
	NO_SPECIAL = `^[^'";$\\]*$`
)

func NewVaC(inParam map[string]string) *verificationConversion {
	return &verificationConversion{
		inParam:  inParam,
		outParam: make(map[string]string),
	}
}

func (this *verificationConversion) Do(ruleObjList map[string]*Rule) *verificationConversion {
	for field, ruleObj := range ruleObjList {
		for _, fn := range ruleObj.fnList {
			err, value := fn(this.inParam[field])
			if err != "" {
				if "nil" != ruleObj.def {
					this.outParam[field] = ruleObj.def
				}
				this.err = lib.StrJoin(field, ":", err)
				return this
			}
			this.outParam[field] = value
		}
	}
	return this
}

func (this *verificationConversion) IsOk() bool {
	return "" != this.err
}

func (this *verificationConversion) Param() map[string]string {
	return this.outParam
}

func (this *verificationConversion) Err() string {
	return this.err
}

//========== rule ==========
type Rule struct {
	def      string
	lenLte   int
	lenGte   int
	regexp   string
	inArr    []string
	inArrInt []int
	inMap    map[string]string

	fnList []func(value string) (string, string)
}

func NewRule() *Rule {
	return &Rule{
		def: "nil",
	}
}

func (this *Rule) LenLte(min int) *Rule {
	this.lenLte = min
	this.fnList = append(this.fnList, func(value string) (string, string) {
		if utf8.RuneCountInString(value) < this.lenLte {
			return "Length is too short.", value
		}
		return "", value
	})
	return this
}

func (this *Rule) LenGte(max int) *Rule {
	this.lenGte = max
	this.fnList = append(this.fnList, func(value string) (string, string) {
		if utf8.RuneCountInString(value) > this.lenGte {
			return "Length is too long.", value
		}
		return "", value
	})
	return this
}

func (this *Rule) RegExp(exp string) *Rule {
	this.regexp = exp
	this.fnList = append(this.fnList, func(value string) (string, string) {
		if ok, _ := regexp.MatchString(this.regexp, value); ok {
			return "", value
		}
		return "Wrong rule.", value
	})
	return this
}

func (this *Rule) InArr(enumList ...string) *Rule {
	this.inArr = enumList
	this.fnList = append(this.fnList, func(value string) (string, string) {
		for _, v := range this.inArr {
			if v == value {
				return "", value
			}
		}
		return "Don't have this item.", value
	})
	return this
}

func (this *Rule) InArrInt(enumList ...int) *Rule {
	this.inArrInt = enumList
	this.fnList = append(this.fnList, func(value string) (string, string) {
		if item, ok := strconv.Atoi(value); ok != nil {
			return "", value
		} else {
			for _, v := range this.inArrInt {
				if v == item {
					return "", value
				}
			}
		}
		return "Don't have this item.", value
	})
	return this
}

func (this *Rule) InMap(mapList map[string]string) *Rule {
	this.inMap = mapList
	this.fnList = append(this.fnList, func(value string) (string, string) {
		if item, ok := this.inMap[value]; ok {
			return "", item
		}
		return "Don't have this item.", value
	})
	return this
}

func (this *Rule) Slash() *Rule {
	this.fnList = append(this.fnList, func(value string) (string, string) {
		return "", strconv.Quote(value)
	})
	return this
}

func (this *Rule) Def(def string) *Rule {
	this.def = def
	return this
}

//========== /rule ==========
