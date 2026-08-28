package http

import (
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/transport/route"
)

// Route describes the HTTP binding of a proto RPC method, used to translate a
// gRPC/rpcx call into a replayable curl against the HTTP endpoint. Generated
// code (protoc-gen-go-svc) emits one entry per method from the google.api.http
// annotation.
type Route = route.Route

// RegisterRoutes registers HTTP route bindings keyed by "Service/Method".
// Generated code calls this at init.
func RegisterRoutes(m map[string]Route) { route.RegisterRoutes(m) }

// LookupRoute returns the HTTP binding for a full method ("Service/Method").
func LookupRoute(fullMethod string) (Route, bool) { return route.LookupRoute(fullMethod) }

// RouterPath converts a route path template from the brace form the HTTP routes
// table uses ("/user/{userId}") to the httprouter colon form ("/user/:userId"),
// which is the proto route template every Transporter's Operation reports.
func RouterPath(tmpl string) string { return route.RouterPath(tmpl) }

// RouteOperation resolves the HTTP binding for a gRPC-style full method
// ("Service/Method"). ok is false when the method has no google.api.http
// binding. op.Operation is the route template in httprouter colon form and
// op.Method the proto HTTP verb; tmpl is the same path in brace form for
// path-param substitution.
func RouteOperation(fullMethod string) (op transport.Operation, tmpl string, ok bool) {
	return route.RouteOperation(fullMethod)
}
