package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/neo532/gofr/ghttp"
)

type Logger struct {
}

func (l *Logger) Log(c context.Context, statusCode int, curl string, limit time.Duration, cost time.Duration, resp []byte, err error) {
	logMsg := fmt.Sprintf("[%s] [code:%+v] [limit:%+v] [cost:%+v] [err:%+v] [%+v]",
		curl,
		statusCode,
		limit,
		cost,
		err,
		string(resp),
	)
	fmt.Println(logMsg)
}

type ReqParam struct {
	Directory string `form:"directory,omitempty"`
}

type Body struct {
	Directory string `json:"directory"`
}

func TestHTTP(t *testing.T) {

	// register logger if it's necessary.
	ghttp.RegLogger(&Logger{})

	// build args
	p := (&ghttp.HTTP{
		Method: "GET",
		URL:    "https://github.com/neo532/gofr",
		Limit:  time.Duration(3) * time.Second, // optional
		Retry:  1,                              // optional, default:1
	}).
		QueryArgs(&ReqParam{Directory: "request"}).                                // optional
		JsonBody(&Body{Directory: "request"}).                                     // optional
		Header(http.Header{"a": []string{"a1", "a2"}, "b": []string{"b1", "b2"}}). // optional
		CheckArgs()

	// check arguments
	if p.Err() != nil {
		fmt.Println(fmt.Sprintf("%s\t:err:%v", t.Name(), p.Err()))
		return
	}

	// request
	p.Request(context.Background())
}
