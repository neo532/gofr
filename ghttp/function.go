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
		err = E_MUST_BE_STRUCT
		return
	}

	var b bytes.Buffer
	for i := 0; i < T.NumField(); i++ {
		field := T.Field(i)
		value := V.Field(i)

		name := field.Tag.Get("form")
		emptyIndex := strings.Index(name, ",omitempty")

		// check if empty,in case of escape to heap,use strings.
		if emptyIndex != -1 {
			if value.IsZero() {
				continue
			}
			name = name[0:emptyIndex]
		}

		// identify type
		switch value.Kind() {
		case reflect.String:
			b = lib.StrBJoin(b, "&", name, "=", url.QueryEscape(value.String()))
		case reflect.Int, reflect.Int64:
			b = lib.StrBJoin(b, "&", name, "=", strconv.FormatInt(value.Int(), 10))
		case reflect.Uint64:
			b = lib.StrBJoin(b, "&", name, "=", strconv.FormatUint(value.Uint(), 10))
		case reflect.Float64:
			b = lib.StrBJoin(b, "&", name, "=", strconv.FormatFloat(value.Float(), 'f', -1, 64))
		case reflect.Slice:
			o := value
			for i, lenS := 0, o.Len(); i < lenS; i++ {
				v := o.Index(i)
				switch v.Kind() {
				case reflect.String:
					b = lib.StrBJoin(b, "&", name, "[]=", url.QueryEscape(v.String()))
				case reflect.Int, reflect.Int64:
					b = lib.StrBJoin(b, "&", name, "[]=", strconv.FormatInt(v.Int(), 10))
				case reflect.Uint64:
					b = lib.StrBJoin(b, "&", name, "[]=", strconv.FormatUint(v.Uint(), 10))
				case reflect.Float64:
					b = lib.StrBJoin(b, "&", name, "[]=", strconv.FormatFloat(v.Float(), 'f', -1, 64))
				default:
					err = E_NOT_SUPPORT_TYPE
					return
				}
			}
		default:
			err = E_NOT_SUPPORT_TYPE
			return
		}
	}
	s = strings.TrimPrefix(b.String(), "&")
	return
}
