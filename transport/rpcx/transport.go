package rpcx

import (
	"github.com/neo532/gofr/transport"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport implements transport.Transporter for rpcx.
type Transport struct {
	endpoint    string
	operation   string
	reqHeader   headerCarrier
	replyHeader headerCarrier
}

func (t *Transport) Kind() transport.Kind           { return transport.KindRPCX }
func (t *Transport) Endpoint() string                { return t.endpoint }
func (t *Transport) Operation() string               { return t.operation }
func (t *Transport) RequestHeader() transport.Header  { return t.reqHeader }
func (t *Transport) ReplyHeader() transport.Header    { return t.replyHeader }

// headerCarrier adapts map[string]string to transport.Header.
type headerCarrier map[string]string

func (h headerCarrier) Get(key string) string { return map[string]string(h)[key] }

func (h headerCarrier) Set(key, value string) {
	map[string]string(h)[key] = value
}

func (h headerCarrier) Keys() []string {
	m := map[string]string(h)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
