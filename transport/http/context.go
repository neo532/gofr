package http

import (
	"context"
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
	ClientIP() string
	Bind(any) error
	Result(int, any) error
	Middleware(transport.Handler) transport.Handler
}

// responseWriter buffers the status code until the first Write, so the
// encoder never needs to know the code: Result records it here and the
// header is only sent once the body is written.
type responseWriter struct {
	code int
	w    http.ResponseWriter
}

func (w *responseWriter) Header() http.Header        { return w.w.Header() }
func (w *responseWriter) WriteHeader(statusCode int) { w.code = statusCode }
func (w *responseWriter) Write(data []byte) (int, error) {
	w.w.WriteHeader(w.code)
	return w.w.Write(data)
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.w }

type wrapper struct {
	req    *http.Request
	res    http.ResponseWriter
	w      responseWriter
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

// ClientIP returns the real client IP, delegating to the request's Transporter.
func (c *wrapper) ClientIP() string {
	tr, _ := transport.FromServerContext(c.req.Context())
	if tr != nil {
		return tr.ClientIP()
	}
	return ""
}

func (c *wrapper) Result(code int, v any) error {
	c.w.WriteHeader(code)
	return c.srv.enc(&c.w, c.req, v)
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
