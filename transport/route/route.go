// Package route holds the shared HTTP-binding registry generated from the
// google.api.http annotations of a proto module: for each RPC method it maps the
// full method name to an HTTP route template and HTTP verb, plus a provider that
// extracts the path-param values from a decoded request message. All gofr
// transports consume it — HTTP to route requests, gRPC/rpcx to report the
// proto route template and real parameterized path in their Operation, and
// WebSocket to stamp route templates. Generated code (protoc-gen-go-svc) emits
// one RegisterRoutes and RegisterPathParams call per proto package at init.
package route

import (
	"strings"
	"sync"

	"github.com/neo532/gofr/transport"
)

// Route describes the HTTP binding of a proto RPC method, used to translate a
// gRPC/rpcx call into a replayable curl against the HTTP endpoint. Generated
// code (protoc-gen-go-svc) emits one entry per method from the google.api.http
// annotation.
type Route struct {
	Service    string // full proto service name, e.g. "user.api.user.UserApi"
	Method     string // RPC method name, e.g. "GetByUserId"
	HTTPMethod string // "GET", "PUT", ...
	Path       string // path template with {param}, e.g. "/user/{userId}"
}

// FullMethod returns the gRPC-style full method name "Service/Method".
func (r Route) FullMethod() string { return r.Service + "/" + r.Method }

var (
	routesMu sync.RWMutex
	routes   = map[string]Route{}
)

// RegisterRoutes registers HTTP route bindings keyed by "Service/Method".
// Generated code calls this at init.
func RegisterRoutes(m map[string]Route) {
	routesMu.Lock()
	defer routesMu.Unlock()
	for k, v := range m {
		routes[k] = v
	}
}

// LookupRoute returns the HTTP binding for a full method ("Service/Method").
func LookupRoute(fullMethod string) (Route, bool) {
	routesMu.RLock()
	defer routesMu.RUnlock()
	r, ok := routes[fullMethod]
	return r, ok
}

// RouterPath converts a route path template from the brace form the route table
// uses ("/user/{userId}") to the httprouter colon form ("/user/:userId"), which
// is the proto route template every Transporter's Operation reports.
func RouterPath(tmpl string) string {
	var b strings.Builder
	b.Grow(len(tmpl))
	in := false
	for i := 0; i < len(tmpl); i++ {
		switch c := tmpl[i]; {
		case c == '{':
			b.WriteByte(':')
			in = true
		case c == '}':
			in = false
		case in:
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// RouteOperation resolves the HTTP binding for a gRPC-style full method
// ("Service/Method"). ok is false when the method has no google.api.http
// binding. op.Operation is the route template in httprouter colon form and
// op.Method the proto HTTP verb; tmpl is the same path in brace form for
// path-param substitution.
func RouteOperation(fullMethod string) (op transport.Operation, tmpl string, ok bool) {
	route, ok := LookupRoute(strings.TrimPrefix(fullMethod, "/"))
	if !ok {
		return transport.Operation{}, "", false
	}
	return transport.Operation{Operation: RouterPath(route.Path), Method: route.HTTPMethod}, route.Path, true
}
