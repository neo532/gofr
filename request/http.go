package request

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
	var p = (&request.HTTP{
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
	p.Do(context.Background())
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// HTTP is Request's HTTPeter.
type HTTP struct {
	queryArgs string

	body     io.Reader
	bodyCurl string

	headerReq  http.Header
	headerCurl string

	err     error
	curl    string
	isCheck bool

	URL    string
	Method string
	Limit  time.Duration
	Retry  int

	cookies []*http.Cookie
}

// QueryArgs deals with form data and returns HTTPself by struct.
func (p *HTTP) QueryArgs(param interface{}) *HTTP {
	var str string
	str, p.err = Struct2QueryArgs(param)
	if p.err != nil || str == "" {
		return p
	}
	p.queryArgs = "?" + str
	return p
}

// OriBody returns HTTPself by origin.
func (p *HTTP) OriBody(param string) *HTTP {
	p.bodyCurl = fmtCurlBody(param)
	p.body = string2ioReader(param)
	return p
}

// JsonBody deals with json data and returns HTTPself by struct.
func (p *HTTP) JsonBody(param interface{}) *HTTP {
	var bytesData []byte
	bytesData, p.err = json.Marshal(param)
	if p.err != nil {
		return p
	}
	p.bodyCurl = fmtCurlBody(string(bytesData))
	p.body = byte2ioReader(bytesData)
	return p
}

// Header returns HTTPself by header.
func (p *HTTP) Header(header http.Header) *HTTP {
	p.headerReq = header

	// log
	var bHeader strings.Builder
	for key, vs := range header {
		for _, value := range vs {
			bHeader.WriteString(fmtCurlOneHeader(key, value))
		}
	}
	p.headerCurl = bHeader.String()
	return p
}

// AddHeader can add one header to HTTP.
func (p *HTTP) AddHeader(key, value string) *HTTP {
	p.headerReq.Add(key, value)
	p.headerCurl += fmtCurlOneHeader(key, value)
	return p
}

// HeaderFollowLocation adds the header for the situation of 302 or 301.
func (p *HTTP) HeaderFollowLocation() *HTTP {
	p.AddHeader("CURLOPT_FOLLOWLOCATION", "TRUE")
	return p
}

// CheckArgs judges that if the HTTP is right.
func (p *HTTP) CheckArgs() *HTTP {
	// check HTTP
	if p.err != nil {
		return p
	}

	switch p.Retry {
	case 0: // default, retry one times
		p.Retry = 1
	case -1: // no retry
		p.Retry = 0
	}
	p.URL = p.URL + p.queryArgs
	p.curl = fmt.Sprintf("curl -X '%s' '%s'%s%s",
		p.Method,
		p.URL,
		p.headerCurl,
		p.bodyCurl,
	)
	p.isCheck = true

	// clean,in case of that the executing step isn't last one.
	p.queryArgs = ""
	p.bodyCurl = ""
	p.headerCurl = ""

	return p
}

func (p *HTTP) Cookies() []*http.Cookie {
	return p.cookies
}

// Err returns the error of HTTP.
func (p *HTTP) Err() error {
	return p.err
}

// Do does a HTTP for multi-times.
func (p *HTTP) Do(c context.Context) (bResp []byte, err error) {
	if p.isCheck == false {
		err = errors.New("Please check!")
		return
	}

	var httpCode int
	for i := 0; i <= p.Retry; i++ {
		bResp, httpCode, err = p.do(c)
		if httpCode == http.StatusOK {
			break
		}
	}
	return
}

// do does with a http.
func (p *HTTP) do(c context.Context) (bResp []byte, statusCode int, err error) {
	// request init
	var req *http.Request
	req, err = http.NewRequest(p.Method, p.URL, p.body)
	if err != nil {
		return
	}

	// header
	req.Header = p.headerReq

	// request
	var client = &http.Client{Timeout: p.Limit}
	var resp *http.Response
	var start = time.Now()
	resp, err = client.Do(req)
	var cost = time.Now().Sub(start)

	// response
	if resp != nil {
		if resp.Body != nil {
			defer resp.Body.Close()
			bResp, err = ioutil.ReadAll(resp.Body)
		}
		p.cookies = resp.Cookies()
		statusCode = resp.StatusCode
	}

	// log
	logger.Log(c, statusCode, p.curl, p.Limit, cost, bResp, err)
	return
}
