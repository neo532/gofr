package http

import "sync"

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
