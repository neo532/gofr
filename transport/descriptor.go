package transport

import (
	"context"
	"fmt"
	"reflect"
)

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, req any) (any, error)

// ValidateServiceMethod checks that a concrete method matches its descriptor at registration time.
func ValidateServiceMethod(svcName, methodName string, method reflect.Value, desc *MethodDesc) {
	if !method.IsValid() {
		panic(fmt.Sprintf("gofr: service %q has no method %q", svcName, methodName))
	}

	t := method.Type()

	// Validate inputs: func(context.Context, *Req)
	if t.NumIn() != 2 {
		panic(fmt.Sprintf("gofr: service %q method %q expects %d input params, want 2 (context.Context, *Req)", svcName, methodName, t.NumIn()))
	}
	if t.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		panic(fmt.Sprintf("gofr: service %q method %q first param must be context.Context, got %s", svcName, methodName, t.In(0)))
	}
	if desc.NewRequest != nil {
		wantReq := reflect.TypeOf(desc.NewRequest())
		if t.In(1) != wantReq {
			panic(fmt.Sprintf("gofr: service %q method %q second param must be %s, got %s", svcName, methodName, wantReq, t.In(1)))
		}
	}

	// Validate outputs: (*Res, error) or just error
	if t.NumOut() == 0 || t.NumOut() > 2 {
		panic(fmt.Sprintf("gofr: service %q method %q returns %d values, want 1 or 2", svcName, methodName, t.NumOut()))
	}
	if t.NumOut() == 2 {
		var errType = reflect.TypeOf((*error)(nil)).Elem()
		if t.Out(1) != errType {
			panic(fmt.Sprintf("gofr: service %q method %q second return must be error, got %s", svcName, methodName, t.Out(1)))
		}
		if desc.NewResponse != nil {
			wantRes := reflect.TypeOf(desc.NewResponse())
			if t.Out(0) != wantRes {
				panic(fmt.Sprintf("gofr: service %q method %q first return must be %s, got %s", svcName, methodName, wantRes, t.Out(0)))
			}
		}
	}
}

// MethodDesc describes a single RPC method.
type MethodDesc struct {
	// Name is the method name, must match the method on the service implementation.
	Name string
	// NewRequest creates a new empty request value.
	NewRequest func() any
	// NewResponse creates a new empty response value. Used for signature validation at registration.
	NewResponse func() any
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
