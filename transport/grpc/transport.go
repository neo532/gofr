package grpc

import (
	"net"

	"google.golang.org/grpc/metadata"

	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/transport/ip"
	"github.com/neo532/gofr/transport/route"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport implements transport.Transporter for gRPC.
type Transport struct {
	endpoint  string
	operation string // proto route template (colon form), or full method when no HTTP binding
	method    string // proto HTTP verb, "" when no HTTP binding
	routeKey  string // "Service/Method", for path-param lookup; "" when no HTTP binding
	pathTmpl  string // brace-form route template, for path-param substitution
	req       any
	params    map[string]string

	reqHeader      headerCarrier
	replyHeader    headerCarrier
	peer           string
	trustedProxies []*net.IPNet
	app            transport.App
}

func (t *Transport) Kind() transport.Kind            { return transport.KindGRPC }
func (t *Transport) Endpoint() string                { return t.endpoint }
func (t *Transport) RequestHeader() transport.Header { return t.reqHeader }
func (t *Transport) ReplyHeader() transport.Header   { return t.replyHeader }
func (t *Transport) App() transport.App              { return t.app }
func (t *Transport) ClientIP() string {
	return ip.ClientIP(t.peer, t.reqHeader.Get, t.trustedProxies)
}

// Operation returns the request identity: the proto route template, the real
// parameterized path (substituted from the decoded request via the generated
// HTTPPathParams provider) and the proto HTTP verb.
func (t *Transport) Operation() transport.Operation {
	o := transport.Operation{Operation: t.operation, Method: t.method}
	if t.req != nil && len(t.params) > 0 && t.pathTmpl != "" {
		if p, ok := route.SubstitutePathParams(t.pathTmpl, t.params); ok {
			o.Path = p
		}
	}
	return o
}

// SetReq records the decoded request so Operation() can resolve the real path.
func (t *Transport) SetReq(req any) {
	t.req = req
	if req != nil && t.routeKey != "" {
		if fn, ok := route.LookupPathParams(t.routeKey); ok {
			t.params = fn(req)
		}
	}
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
