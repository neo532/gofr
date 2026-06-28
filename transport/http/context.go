package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// Context is an HTTP request context.
type Context interface {
	context.Context
	Request() *http.Request
	Response() http.ResponseWriter
	PathValue(key string) string
	Query() url.Values
	Bind(any) error
	JSON(int, any) error
	String(int, string) error
	Result(int, any) error
	Middleware(transport.Handler) transport.Handler
}

type wrapper struct {
	req    *http.Request
	res    http.ResponseWriter
	srv    *Server
	codec  DecodeRequestFunc
	params httprouter.Params
}

var _ Context = (*wrapper)(nil)

func (c *wrapper) Deadline() (time.Time, bool)    { return c.req.Context().Deadline() }
func (c *wrapper) Done() <-chan struct{}           { return c.req.Context().Done() }
func (c *wrapper) Err() error                      { return c.req.Context().Err() }
func (c *wrapper) Value(k any) any { return c.req.Context().Value(k) }
func (c *wrapper) Request() *http.Request          { return c.req }
func (c *wrapper) Response() http.ResponseWriter   { return c.res }
func (c *wrapper) PathValue(key string) string     { return c.params.ByName(key) }
func (c *wrapper) Query() url.Values               { return c.req.URL.Query() }
func (c *wrapper) Bind(v any) error        { return c.codec(c.req, v) }

func (c *wrapper) JSON(code int, v any) error {
	c.res.Header().Set("Content-Type", "application/json")
	c.res.WriteHeader(code)
	return json.NewEncoder(c.res).Encode(v)
}

func (c *wrapper) String(code int, text string) error {
	c.res.Header().Set("Content-Type", "text/plain")
	c.res.WriteHeader(code)
	_, err := io.WriteString(c.res, text)
	return err
}

func (c *wrapper) Result(code int, v any) error {
	return c.srv.enc(c.res, c.req, v)
}

func (c *wrapper) Middleware(userHandler transport.Handler) transport.Handler {
	tr, _ := transport.FromServerContext(c.req.Context())
	op := ""
	if tr != nil {
		op = tr.Operation()
	}
	matched := c.srv.mwManager.Match(op)
	return middleware.Chain(matched...)(userHandler)
}
