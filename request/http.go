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
	var c = context.Background()
	var p = request.Param{}.
		Form(&ReqParam{Directory: "request"}).
		Json(&Body{Directory: "request"})
	request.Request(c,
		time.Duration(2)*time.Second,
		"GET",
		"https://github.com/neo532/gofr",
		p,
		nil,
	)
*/

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ========== logger ==========
var logger Logger = &LoggerDefault{}

// Logger is a interface for Log.
type Logger interface {
	// Log's situation:
	// Timeout,if cost>limit,
	// StatusCode is bad,if statusCode!=http.StatusOK
	Log(c context.Context, statusCode int, curl string, limit time.Duration, cost time.Duration, resp []byte, err error)
}

// InitLogger is a register at starup.
func InitLogger(l Logger) {
	logger = l
}

// LoggerDefault is a default value for logger.
type LoggerDefault struct {
}

// Log is a default value to logger for showing.
func (l *LoggerDefault) Log(c context.Context, statusCode int, curl string, limit time.Duration, cost time.Duration, resp []byte, err error) {
	var logMsg = fmt.Sprintf("[%s] [code:%+v] [limit:%+v] [cost:%+v] [%+v]",
		curl,
		statusCode,
		limit,
		cost,
		string(resp),
	)
	fmt.Println(logMsg)
	return
}

// ========== Param ==========
// Param is Request's paramter.
type Param struct {
	BodyLog string
	ReqArgs string
	Body    io.Reader
	Err     error
	Retry   int
}

// Ori returns Paramself by origin.
func (p Param) Ori(param string) Param {
	p.BodyLog = " -d " + strconv.Quote(param)
	p.Body = p.string2ReqParam(param)
	return p
}

// Json deals with json data and returns Paramself by struct.
func (p Param) Json(param interface{}) Param {
	var bytesData []byte
	bytesData, p.Err = json.Marshal(param)
	if p.Err != nil {
		return p
	}
	p.BodyLog = " -d " + "'" + strings.Trim(strconv.Quote(string(bytesData)), `"`) + "'"
	p.Body = p.byte2ReqParam(bytesData)
	return p
}

// Form deals with form data and returns Paramself by struct.
func (p Param) Form(param interface{}) Param {
	var str string
	str, p.Err = p.Struct2String(param)
	if p.Err != nil {
		return p
	}
	p.ReqArgs = "?" + str
	return p
}

// Struct2String turn the struct data to string data.
func (p Param) Struct2String(param interface{}) (r string, err error) {
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

// string2ReqParam turns the string data to body.
func (p Param) string2ReqParam(param string) io.Reader {
	var body *strings.Reader
	body = strings.NewReader(param)
	return body
}

// byte2ReqParam turns the bytes data to body.
func (p Param) byte2ReqParam(param []byte) io.Reader {
	var body io.Reader
	body = bytes.NewReader(param)
	return body
}

// ========== Header ==========
// Header is a object of request header.
type Header map[string]string

// FollowLocation adds the header for the situation of 302 or 301.
func (h Header) FollowLocation() Header {
	h["CURLOPT_FOLLOWLOCATION"] = "TRUE"
	return h
}

// ========== Request ==========
// Request does a request for multi-times.
func Request(
	c context.Context,
	timeOut time.Duration,
	method string,
	url string,
	param Param,
	header map[string]string,
) (bResp []byte, err error) {

	// check param
	if param.Err != nil {
		err = param.Err
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
		bResp, httpCode, err = request(c, timeOut, method, url, param, header)
		if httpCode == http.StatusOK {
			break
		}
	}
	return
}

// request does with a request.
func request(
	c context.Context,
	timeOut time.Duration,
	method string,
	url string,
	param Param,
	header map[string]string,
) (bResp []byte, statusCode int, err error) {

	// check param
	if param.Err != nil {
		err = param.Err
		return
	}

	// request init
	var req *http.Request
	var reqFull = url + param.ReqArgs
	req, err = http.NewRequest(method, reqFull, param.Body)
	if err != nil {
		return
	}

	// header
	var bHeader strings.Builder
	if header != nil {
		for k, v := range header {
			req.Header.Add(k, v)
			bHeader.WriteString(" -H '" + k + ":" + v + "'")
		}
	}

	// request
	var client = &http.Client{Timeout: timeOut}
	var resp *http.Response
	startT := time.Now()
	resp, err = client.Do(req)
	endT := time.Now()
	cost := endT.Sub(startT)

	// response
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
	}
	if err == nil && resp != nil {
		bResp, err = ioutil.ReadAll(resp.Body)
	}

	// log
	var curl = fmt.Sprintf("curl -X '%s' '%s'%s%s",
		method,
		reqFull,
		bHeader.String(),
		param.BodyLog,
	)
	logger.Log(c, statusCode, curl, timeOut, cost, bResp, err)
	return
}
