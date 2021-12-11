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

	"github.com/neo532/gofr/lib"
	"github.com/neo532/gofr/lib/slices"
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
		err = fmt.Errorf("%v must be a struct or a struct pointer", param)
		return
	}

	var b bytes.Buffer
	for i := 0; i < T.NumField(); i++ {
		field := T.Field(i)
		value := V.Field(i)
		var tag slices.String = strings.Split(field.Tag.Get("form"), ",")

		// check if empty
		if slices.In("omitempty", tag) && value.IsZero() {
			continue
		}

		// identify type
		if b, err = reflectKind2Byte(b, tag, value, "=", nil); err != nil {
			return
		}
	}
	s = strings.TrimPrefix(b.String(), "&")
	return
}

// reflectKind2Byte turns the condition to bytes by reflect.
func reflectKind2Byte(b bytes.Buffer, tag []string, value reflect.Value, equal string, err error) (bytes.Buffer, error) {
	if err != nil {
		return b, err
	}
	switch value.Kind() {
	case reflect.String:
		b = lib.StrBJoin(b, "&", tag[0], equal, url.QueryEscape(value.String()))
	case reflect.Int, reflect.Int64:
		b = lib.StrBJoin(b, "&", tag[0], equal, strconv.FormatInt(value.Int(), 10))
	case reflect.Uint64:
		b = lib.StrBJoin(b, "&", tag[0], equal, strconv.FormatUint(value.Uint(), 10))
	case reflect.Float64:
		b = lib.StrBJoin(b, "&", tag[0], equal, strconv.FormatFloat(value.Float(), 'f', -1, 64))
	case reflect.Slice:
		o := value
		lenS := o.Len()
		if lenS > 0 {
			for i := 0; i < lenS; i++ {
				b, err = reflectKind2Byte(b, tag, o.Index(i), "[]=", err)
			}
		}
	default:
		err = fmt.Errorf(
			"%v isn't support type. string/int/int64/uint64/float64/[]string/[]int/[]int64/[]uint64/[]float64 only",
			value.Kind(),
		)
		return b, err
	}
	return b, nil
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

func fmtCurlBody(body string) (str string) {
	if body == "" {
		return
	}
	return " -d " + "'" + strings.Trim(strconv.Quote(body), `"`) + "'"
}
