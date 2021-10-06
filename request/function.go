package request

/*
 * @abstract functions
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Struct2ReqArgs turn the struct data to string data.
func Struct2ReqArgs(param interface{}) (r string, err error) {

	// unify type
	T := reflect.TypeOf(param)
	V := reflect.ValueOf(param)
	switch {
	case T.Kind() == reflect.Struct:
	case T.Kind() == reflect.Ptr && T.Elem().Kind() == reflect.Struct:
		T = T.Elem()
		V = V.Elem()
	default:
		err = fmt.Errorf("%v must be a struct or a struct pointer", param)
		return
	}

	var b strings.Builder
	for i := 0; i < T.NumField(); i++ {
		var field = T.Field(i)
		var name = field.Name
		var tag = strings.Split(field.Tag.Get("form"), ",")
		var objField = V.Field(i)

		// check if empty
		if len(tag) > 1 {
			isBreak := false
			for _, v := range tag[1:] {
				if v == "omitempty" {
					isBreak = true
					break
				}
			}
			if isBreak == true && objField.IsZero() {
				continue
			}
		}

		b.WriteString("&")
		b.WriteString(tag[0])
		b.WriteString("=")
		switch objField.Kind() {
		case reflect.String:
			b.WriteString(url.QueryEscape(V.FieldByName(name).String()))
		case reflect.Float64:
			b.WriteString(strconv.FormatFloat(V.FieldByName(name).Float(), 'f', -1, 64))
		case reflect.Int, reflect.Int64:
			b.WriteString(strconv.FormatInt(V.FieldByName(name).Int(), 10))
		case reflect.Uint64:
			b.WriteString(strconv.FormatUint(V.FieldByName(name).Uint(), 10))
		default:
			err = fmt.Errorf(
				"%v isnot support type. string/float64/int64/int only",
				objField.Kind(),
			)
			return
		}
	}
	r = strings.TrimPrefix(b.String(), "&")
	return
}

// string2ioReader turns the string data to io.Reader.
func string2ioReader(param string) io.Reader {
	var ioReader *strings.Reader
	ioReader = strings.NewReader(param)
	return ioReader
}

// byte2ioReader turns the bytes data to io.Reader.
func byte2ioReader(param []byte) io.Reader {
	var ioReader io.Reader
	ioReader = bytes.NewReader(param)
	return ioReader
}

func fmtCurlOneHeader(key, value string) string {
	return " -H '" + key + ":" + value + "'"
}

func fmtCurlBody(body string) string {
	return " -d " + "'" + strings.Trim(strconv.Quote(body), `"`) + "'"
}
