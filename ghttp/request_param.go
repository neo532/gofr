package ghttp

/*
 * @abstract http request
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-02
 * @demo:
	type Req struct {
		Directory string `form:"directory"`
	}
	type Body struct {
		Directory string `json:"directory"`
	}
	p := (&ghttp.HTTP{
		Limit: time.Duration(3)*time.Second,
		URL: "https://github.com/neo532/gofr",
		Method: "GET",
	}).
		QueryArgs(&Req{Directory: "request"}).
		JsonBody(&Body{Directory: "request"}).
		Header(http.Header{"a": []string{"a1", "a2"}, "b":[]string{"b1", "b2"}}).
		CheckArgs()
	if p.Err() != nil {
		fmt.Println(p.Err())
		return
	}
	p.Request(context.Background())
*/

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Param is Request's HTTPeter.
type Param struct {
	Body     io.Reader
	BodyCurl string

	URL    string
	Method string
	Limit  time.Duration
	Retry  int
}

// OriBody returns HTTPself by origin.
func OriBody(param string) (r io.Reader, l string) {
	if param != "" {
		l = " -d " + "'" + strings.Trim(strconv.Quote(param), `"`) + "'"
	}
	r = strings.NewReader(param)
	return
}

// JsonBody deals with json data and returns HTTPself by struct.
func JsonBody(param interface{}) (r io.Reader, l string, err error) {
	var bytesData []byte
	bytesData, err = json.Marshal(param)
	if err != nil {
		return
	}
	if param != "" {
		l = " -d " + "'" + strings.Trim(strconv.Quote(string(bytesData)), `"`) + "'"
	}
	r = bytes.NewReader(bytesData)
	return
}

// HeaderLog returns HTTPself by header.
func HeaderLog(header http.Header) (l string) {
	// log
	var bHeader strings.Builder
	for key, vs := range header {
		for _, value := range vs {
			bHeader.WriteString(" -H '" + key + ":" + value + "'")
		}
	}
	l = bHeader.String()
	return
}
