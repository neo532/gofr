package grpc

import (
	"net"

	"google.golang.org/grpc/metadata"

	"github.com/neo532/gofr/transport"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport implements transport.Transporter for gRPC.
type Transport struct {
	endpoint       string
	operation      string
	reqHeader      headerCarrier
	replyHeader    headerCarrier
	peer           string
	trustedProxies []*net.IPNet
	app            transport.App
}

func (t *Transport) Kind() transport.Kind            { return transport.KindGRPC }
func (t *Transport) Endpoint() string                { return t.endpoint }
func (t *Transport) Operation() string               { return t.operation }
func (t *Transport) RequestHeader() transport.Header { return t.reqHeader }
func (t *Transport) ReplyHeader() transport.Header   { return t.replyHeader }
func (t *Transport) App() transport.App              { return t.app }
func (t *Transport) ClientIP() string {
	return transport.ClientIP(t.peer, t.reqHeader.Get, t.trustedProxies)
}

// headerCarrier adapts metadata.MD to transport.Header.
type headerCarrier metadata.MD

func (h headerCarrier) Get(key string) string {
	vals := metadata.MD(h).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (h headerCarrier) Set(key, value string) {
	metadata.MD(h).Set(key, value)
}

func (h headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range metadata.MD(h) {
		keys = append(keys, k)
	}
	return keys
}
