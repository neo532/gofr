package http

import (
	"github.com/neo532/gofr/transport/route"
)

// PathParamFunc returns a request message's path-param values as strings,
// keyed by path template name. Generated code (protoc-gen-go-svc) emits one
// closure per method so a gRPC/rpcx call can be replayed as an HTTP curl
// without protobuf reflection.
type PathParamFunc = route.PathParamFunc

// RegisterPathParams registers path-param providers keyed by "Service/Method".
// Generated code calls this at init.
func RegisterPathParams(m map[string]PathParamFunc) { route.RegisterPathParams(m) }

// LookupPathParams returns the path-param provider for a full method
// ("Service/Method").
func LookupPathParams(fullMethod string) (PathParamFunc, bool) {
	return route.LookupPathParams(fullMethod)
}

// SubstitutePathParams replaces {param} placeholders in a brace-form route path
// template with values from params. ok is false when a parameter is missing or
// empty, so callers can treat the template as unresolvable.
func SubstitutePathParams(tmpl string, params map[string]string) (string, bool) {
	return route.SubstitutePathParams(tmpl, params)
}
