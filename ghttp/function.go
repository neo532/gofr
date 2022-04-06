package ghttp

/*
 * @abstract functions
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

import (
	"bytes"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/neo532/gofr/lib"
)

// Form's delimiter.
const (
	FORM_AND          = "&"
	FORM_ASSIGN       = "="
	FORM_ASSIGN_SLICE = "[]="
)
var TagName = "form"

// Struct2QueryArgs turn the struct data to string data.
func Struct2QueryArgs(param interface{}) (s string, err error) {

	// unify type
	T := reflect.TypeOf(param)
	V := reflect.ValueOf(param)
	switch {
	case T.Kind() == reflect.Struct:
	case T.Kind() == reflect.Ptr && T.Elem().Kind() == reflect.Struct:
		T = T.Elem()
		V = V.Elem()
	default:
		err = ErrMustBeStruct
		return
	}

	var b bytes.Buffer
	for i := 0; i < T.NumField(); i++ {
		field := T.Field(i)
		if field.PkgPath != "" && !field.Anonymous { // unexported
			continue
		}
		value := V.Field(i)

		name := field.Tag.Get(TagName)
		// don't parse that the name is -.
		if name == "-" {
			continue
		}

		// check whether if empty,in case of escape to heap,use strings.
		emptyIndex := strings.Index(name, ",omitempty")
		if emptyIndex != -1 {
			if value.IsZero() {
				continue
			}
			name = name[0:emptyIndex]
		}

		// identify type
		switch value.Kind() {
		case reflect.String:
			b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN, url.QueryEscape(value.String()))
		case reflect.Int, reflect.Int64:
			b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN, strconv.FormatInt(value.Int(), 10))
		case reflect.Uint64:
			b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN, strconv.FormatUint(value.Uint(), 10))
		case reflect.Float64:
			b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN, strconv.FormatFloat(value.Float(), 'f', -1, 64))
		case reflect.Bool:
			b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN, strconv.FormatBool(value.Bool()))
		case reflect.Slice, reflect.Array:
			o := value
			for i, lenS := 0, o.Len(); i < lenS; i++ {
				v := o.Index(i)
				switch v.Kind() {
				case reflect.String:
					b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN_SLICE, url.QueryEscape(v.String()))
				case reflect.Int, reflect.Int64:
					b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN_SLICE, strconv.FormatInt(v.Int(), 10))
				case reflect.Uint64:
					b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN_SLICE, strconv.FormatUint(v.Uint(), 10))
				case reflect.Float64:
					b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN_SLICE, strconv.FormatFloat(v.Float(), 'f', -1, 64))
				case reflect.Bool:
					b = lib.StrBJoin(b, FORM_AND, name, FORM_ASSIGN_SLICE, strconv.FormatBool(v.Bool()))
				default:
					err = ErrNotSupportType
					return
				}
			}
		default:
			err = ErrNotSupportType
			return
		}
	}
	s = strings.TrimPrefix(b.String(), "&")
	return
}
