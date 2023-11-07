package unicode

import (
	"strconv"
	"strings"
)

type Unicode struct {
	delimiter string
}

type opt func(o *Unicode)

func WithDelimiter(d byte) opt {
	return func(o *Unicode) {
		o.delimiter = string(d)
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
	str := []rune(string(origin))
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
	l := len(en)
	var r strings.Builder

	var i int
	for i < l {
		c := string(en[i])
		if c != o.delimiter {
			i++
			r.WriteString(c)
			continue
		}
		if i+5 <= l {
			cu := string(en[i+1 : i+5])
			i += 5
			var s string
			if s, err = strconv.Unquote(`"\u` + cu + `"`); err != nil {
				return
			}
			r.WriteString(s)
		}
	}
	origin = []byte(r.String())
	return
}
