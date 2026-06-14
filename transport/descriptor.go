package transport

import "context"

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, req interface{}) (interface{}, error)

// MethodDesc describes a single RPC method.
type MethodDesc struct {
	// Name is the method name, must match the method on the service implementation.
	Name string
	// NewRequest creates a new empty request value.
	NewRequest func() interface{}
	// HTTPMethod is optional HTTP method override (e.g. "GET", "POST").
	HTTPMethod string
	// HTTPPath is optional HTTP path override (e.g. "/api/v1/hello/{name}").
	HTTPPath string
}

// ServiceDesc describes a service and its methods.
// It is protocol-agnostic — generated once, used by all transports.
type ServiceDesc struct {
	Name    string
	Methods []MethodDesc
}
