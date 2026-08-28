package route

import (
	"strings"
	"sync"
)

// PathParamFunc returns a request message's path-param values as strings,
// keyed by path template name. Generated code (protoc-gen-go-svc) emits one
// closure per method so a gRPC/rpcx call can be replayed as an HTTP curl
// without protobuf reflection.
type PathParamFunc func(any) map[string]string

var (
	pathParamsMu sync.RWMutex
	pathParams   = map[string]PathParamFunc{}
)

// RegisterPathParams registers path-param providers keyed by "Service/Method".
// Generated code calls this at init.
func RegisterPathParams(m map[string]PathParamFunc) {
	pathParamsMu.Lock()
	defer pathParamsMu.Unlock()
	for k, v := range m {
		pathParams[k] = v
	}
}

// LookupPathParams returns the path-param provider for a full method
// ("Service/Method").
func LookupPathParams(fullMethod string) (PathParamFunc, bool) {
	pathParamsMu.RLock()
	defer pathParamsMu.RUnlock()
	f, ok := pathParams[fullMethod]
	return f, ok
}

// SubstitutePathParams replaces {param} placeholders in a brace-form route path
// template with values from params. ok is false when a parameter is missing or
// empty, so callers can treat the template as unresolvable.
func SubstitutePathParams(tmpl string, params map[string]string) (string, bool) {
	out := tmpl
	for {
		start := strings.Index(out, "{")
		if start < 0 {
			return out, true
		}
		end := strings.Index(out[start:], "}")
		if end < 0 {
			return "", false
		}
		name := out[start+1 : start+end]
		val, ok := params[name]
		if !ok || val == "" {
			return "", false
		}
		out = out[:start] + val + out[start+end+1:]
	}
}
