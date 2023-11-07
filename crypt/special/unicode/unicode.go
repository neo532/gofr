package unicode

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrContainDelimiter = errors.New("Please input a string without delimiter")
)

type Unicode struct {
	delimiter string
}

type opt func(o *Unicode)

func WithDelimiter(d string) opt {
	return func(o *Unicode) {
		o.delimiter = d
	}
}

func New(opts ...opt) (o *Unicode) {
	o = &Unicode{
		delimiter: ".",
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

func (o *Unicode) Encrypt(origin []byte) (en string, err error) {
	ori := string(origin)

	if strings.Contains(ori, o.delimiter) {
		err = ErrContainDelimiter
		return
	}

	str := []rune(ori)
	l := len(str)
	var r strings.Builder

	for i := 0; i < l; i++ {
		s := strconv.QuoteRuneToASCII(str[i])
		r.WriteString(strings.NewReplacer("\\u", o.delimiter, "'", "").Replace(s))
	}

	en = r.String()
	return
}

func (o *Unicode) Decrypt(en string) (origin []byte, err error) {
	var r strings.Builder

	itemS := strings.SplitAfter(en, o.delimiter)

	var has bool
	for _, item := range itemS {

		if has == true {
			var s string
			if s, err = strconv.Unquote(`"\u` + string(item[:4]) + `"`); err != nil {
				return
			}
			r.WriteString(s)
			if len(item) >= 5 {
				item = string(item[4:])
			}
			has = false
		}

		v := strings.TrimSuffix(item, o.delimiter)
		if v != item {
			has = true
			r.WriteString(v)
			continue
		}

		r.WriteString(item)
	}

	origin = []byte(r.String())
	return
}
