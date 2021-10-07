package request

/*
 * @abstract http request
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-02
 * @demo:
	type ReqParam struct {
		Directory string `form:"directory"`
	}
	type Body struct {
		Directory string `json:"directory"`
	}
	var p = request.Param{Limit: time.Duration(3)*time.Second}.
		QueryArgs(&ReqParam{Directory: "request"}).
		JsonBody(&Body{Directory: "request"}).
		Header(http.Header{"a": []string{"a1", "a2"}, "b":[]string{"b1", "b2"}})
	request.Request(context.Background(), "GET", "https://github.com/neo532/gofr", p)
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Param is Request's parameter.
type Param struct {
	queryArgs string

	body     io.Reader
	bodyCurl string

	headerReq  http.Header
	headerCurl string

	err error

	Retry int
	Limit time.Duration
}

// Header returns Paramself by header.
func (p Param) Header(header http.Header) Param {
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

// AddHeader can add one header to Param.
func (p Param) AddHeader(key, value string) Param {
	p.headerReq.Add(key, value)
	p.headerCurl += fmtCurlOneHeader(key, value)
	return p
}

// HeaderFollowLocation adds the header for the situation of 302 or 301.
func (p Param) HeaderFollowLocation() Param {
	p.AddHeader("CURLOPT_FOLLOWLOCATION", "TRUE")
	return p
}

// OriBody returns Paramself by origin.
func (p Param) OriBody(param string) Param {
	p.bodyCurl = fmtCurlBody(param)
	p.body = string2ioReader(param)
	return p
}

// JsonBody deals with json data and returns Paramself by struct.
func (p Param) JsonBody(param interface{}) Param {
	var bytesData []byte
	bytesData, p.err = json.Marshal(param)
	if p.err != nil {
		return p
	}
	p.bodyCurl = fmtCurlBody(string(bytesData))
	p.body = byte2ioReader(bytesData)
	return p
}

// QueryArgs deals with form data and returns Paramself by struct.
func (p Param) QueryArgs(param interface{}) Param {
	var str string
	str, p.err = Struct2QueryArgs(param)
	if p.err != nil {
		return p
	}
	p.queryArgs = "?" + str
	return p
}

// DoHTTP does a HTTP for multi-times.
func DoHTTP(
	c context.Context,
	method string,
	url string,
	param Param,
) (bResp []byte, err error) {

	// check param
	if param.err != nil {
		err = param.err
		return
	}

	var httpCode int
	switch param.Retry {
	case 0: // default, retry one times
		param.Retry = 1
	case -1: // no retry
		param.Retry = 0
	}

	for i := 0; i <= param.Retry; i++ {
		bResp, httpCode, err = doHTTP(c, method, url, param)
		if httpCode == http.StatusOK {
			break
		}
	}
	return
}

// http does with a http.
func doHTTP(
	c context.Context,
	method string,
	url string,
	param Param,
) (bResp []byte, statusCode int, err error) {

	// check param
	if param.err != nil {
		err = param.err
		return
	}

	// request init
	var req *http.Request
	var reqFull = url + param.queryArgs
	req, err = http.NewRequest(method, reqFull, param.body)
	if err != nil {
		return
	}

	// header
	req.Header = param.headerReq

	// request
	var client = &http.Client{Timeout: param.Limit}
	var resp *http.Response
	var start = time.Now()
	resp, err = client.Do(req)
	var cost = time.Now().Sub(start)

	// response
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		if err == nil {
			bResp, err = ioutil.ReadAll(resp.Body)
		}
	}

	// log
	var curl = fmt.Sprintf("curl -X '%s' '%s'%s%s",
		method,
		reqFull,
		param.headerCurl,
		param.bodyCurl,
	)
	logger.Log(c, statusCode, curl, param.Limit, cost, bResp, err)
	return
}
