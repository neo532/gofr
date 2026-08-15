package http

import (
	"net"
	"net/http"

	"github.com/neo532/gofr/transport"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport implements transport.Transporter for HTTP.
type Transport struct {
	endpoint       string
	operation      string
	reqHeader      headerCarrier
	replyHeader    headerCarrier
	peer           string
	trustedProxies []*net.IPNet
	app            transport.App
	req            *http.Request
}

func (t *Transport) Kind() transport.Kind            { return transport.KindHTTP }
func (t *Transport) Endpoint() string                 { return t.endpoint }
func (t *Transport) Operation() string                { return t.operation }
func (t *Transport) RequestHeader() transport.Header  { return t.reqHeader }
func (t *Transport) ReplyHeader() transport.Header    { return t.replyHeader }
func (t *Transport) App() transport.App               { return t.app }

// RawRequest returns the underlying *http.Request. Middleware that needs the
// method, real URL or query string (not available from Operation(), which holds
// the route template) can reach it here.
func (t *Transport) RawRequest() *http.Request { return t.req }
func (t *Transport) ClientIP() string {
	return transport.ClientIP(t.peer, t.reqHeader.Get, t.trustedProxies)
}

type headerCarrier http.Header

func (h headerCarrier) Get(key string) string  { return http.Header(h).Get(key) }
func (h headerCarrier) Set(key, value string)  { http.Header(h).Set(key, value) }
func (h headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range http.Header(h) {
		keys = append(keys, k)
	}
	return keys
}
