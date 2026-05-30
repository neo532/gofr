package http

import (
	"net/http"

	"github.com/neo532/gofr/transport"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport implements transport.Transporter for HTTP.
type Transport struct {
	endpoint    string
	operation   string
	reqHeader   headerCarrier
	replyHeader headerCarrier
}

func (t *Transport) Kind() transport.Kind           { return transport.KindHTTP }
func (t *Transport) Endpoint() string                { return t.endpoint }
func (t *Transport) Operation() string               { return t.operation }
func (t *Transport) RequestHeader() transport.Header  { return t.reqHeader }
func (t *Transport) ReplyHeader() transport.Header    { return t.replyHeader }

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
