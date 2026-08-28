package rpcx

import (
	"net"

	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/transport/ip"
	"github.com/neo532/gofr/transport/route"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport implements transport.Transporter for rpcx.
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
	// onReply, when set by middleware, is invoked with the real reply once the
	// RPC method completes (see middlewarePlugin.PostCall).
	onReply func(reply any, err error)
}

// OnReply registers a handler to observe the real reply after the method runs.
// rpcx runs the middleware chain in PreCall, before the method executes, so the
// chain only sees the decoded request; the reply becomes available in PostCall.
func (t *Transport) OnReply(h func(reply any, err error)) { t.onReply = h }

func (t *Transport) fireReply(reply any, err error) {
	if t.onReply != nil {
		t.onReply(reply, err)
	}
}

func (t *Transport) Kind() transport.Kind           { return transport.KindRPCX }
func (t *Transport) Endpoint() string                { return t.endpoint }
func (t *Transport) RequestHeader() transport.Header  { return t.reqHeader }
func (t *Transport) ReplyHeader() transport.Header    { return t.replyHeader }
func (t *Transport) App() transport.App              { return t.app }

// Operation returns the request identity: the proto route template, the real
// parameterized path (substituted from the decoded request via the generated
// HTTPPathParams provider) and the proto HTTP verb.
func (t *Transport) Operation() transport.Operation {
	o := transport.Operation{Operation: t.operation, Method: t.method}
	if t.req != nil && t.pathTmpl != "" {
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

// ClientIP resolves the real client IP from rpcx request metadata
// (X-Forwarded-For / X-Real-IP) set by the client, honoring the server's
// TrustedProxies configuration and falling back to the direct TCP peer.
func (t *Transport) ClientIP() string {
	return ip.ClientIP(t.peer, t.reqHeader.Get, t.trustedProxies)
}

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
