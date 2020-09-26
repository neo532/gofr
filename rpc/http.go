/*
 * @abstract http request
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 * demo : string(
	 rpc.NewHttp().
		 Header(map[string][string]{"Content-Type":"text/html;"}).
		 Get("https://github.com")
 )
*/
package rpc

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type req struct {
	url     string
	method  string
	retry   int
	timeout time.Duration
	header  map[string]string
	body    string

	//userAgent' => 'Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36',
	//isGzip' => true,
	//referer' => '',
	//cookieList' => array(),
	//proxy' => '',
}
type resp struct {
	header   *http.Response
	body     []byte
	succCode map[int]bool
	err      error
}

type Request struct {
	//log *log
	req  req
	resp resp
}

func NewHttp() *Request {
	return &Request{
		req: req{
			timeout: 3 * time.Second,
			retry:   2,
			body:    "",
		},
		resp: resp{
			succCode: map[int]bool{200: true},
		},
	}
}

func (r *Request) Header(pl map[string]string) *Request {
	r.req.header = pl
	return r
}

//单位秒
func (r *Request) Timeout(t time.Duration) *Request {
	r.req.timeout = t
	return r
}
func (r *Request) SuccCode(codeList []int) *Request {
	for _, v := range codeList {
		r.resp.succCode[v] = true
	}
	return r
}
func (r *Request) Retry(times int) *Request {
	return r
}
func (r *Request) Cookie(times int) *Request {
	return r
}
func (r *Request) Get(url string) []byte {
	r.req.url = url
	r.req.method = "GET"
	r.Do()
	return r.resp.body
}
func (r *Request) Post(url string, param string) interface{} {
	r.req.url = url
	r.req.method = "POST"
	r.req.body = param
	r.Do()
	return r.resp.body
}
func (r *Request) IsOk() bool {
	if nil != r.resp.err {
		return false
	}
	if _, ok := r.resp.succCode[r.resp.header.StatusCode]; ok {
		return true
	}
	return false
}

//遵循RFC 3986编码，空格会被百分号编码(%20)
func Map2url(param map[string]string) string {
	query := make(url.Values)
	for k, v := range param {
		query.Add(k, v)
	}
	return query.Encode()
}

func (r *Request) Do() *Request {
	client := &http.Client{Timeout: r.req.timeout}
	reader := strings.NewReader(r.req.body)
	req, err := http.NewRequest(r.req.method, r.req.url, &Reader{Reader: reader, Offset: 0})
	if err != nil {
		r.resp.err = err
		return r
	}

	//set header
	for field, value := range r.req.header {
		req.Header.Add(field, value)
	}

	//request
	for i := 0; i < r.req.retry; i++ {
		r.resp.header, r.resp.err = client.Do(req)
		if true == r.IsOk() {
			req.Body = NewReader(reader)
			r.resp.body, _ = ioutil.ReadAll(r.resp.header.Body)
			break
		}
	}
	if nil != r.resp.header && nil != r.resp.header.Body {
		r.resp.header.Body.Close()
	}

	return r
}

//========== use for post body ==========
type Reader struct {
	Reader io.ReaderAt
	Offset int64
}

func (p *Reader) Read(val []byte) (n int, err error) {
	n, err = p.Reader.ReadAt(val, p.Offset)
	p.Offset += int64(n)
	return
}

func (p *Reader) Close() error {
	if rc, ok := p.Reader.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}
func NewReader(reader io.ReaderAt) *Reader {
	return &Reader{
		Reader: reader,
		Offset: 0,
	}
}

//========== /use for post body ==========
