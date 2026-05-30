package http

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// RegisterService registers all methods of a service via ServiceDesc.
// Reflection happens only at registration; the middleware chain is also pre-built at registration
// for zero allocation at request time.
// If MethodDesc contains HTTPMethod/HTTPPath (generated from proto google.api.http annotations),
// the route is registered using the annotated method+path; otherwise it falls back to POST /{ServiceName}/{MethodName}.
func RegisterService(s *Server, desc *transport.ServiceDesc, svr interface{}) {
	val := reflect.ValueOf(svr)

	for _, m := range desc.Methods {
		method := val.MethodByName(m.Name)
		if !method.IsValid() {
			panic(fmt.Sprintf("gofr: service %q has no method %q", desc.Name, m.Name))
		}

		// Determine HTTP method + path from annotation, fall back to defaults
		httpMethod := m.HTTPMethod
		if httpMethod == "" {
			httpMethod = "POST"
		}

		path := m.HTTPPath
		if path == "" {
			path = fmt.Sprintf("/%s/%s", desc.Name, m.Name)
		} else {
			path = toHTTPRouterPath(path)
		}

		operation := path
		// Pre-extract path parameter names for injection at request time
		paramNames := extractPathParams(m.HTTPPath)

		fastHandler := buildFastHandler(method)

		// ★ Pre-build middleware chain at registration to avoid Match + Chain closure allocation at request time
		matched := s.mwManager.Match(operation)
		prebuilt := middleware.Chain(matched...)(fastHandler)

		s.Handle(httpMethod, path, func(ctx Context) error {
			req := m.NewRequest()

			if err := ctx.Bind(req); err != nil {
				return err
			}

			// Inject fields from path parameters (supports {name} → Name, etc.)
			injectPathParams(req, paramNames, ctx)

			out, err := prebuilt(ctx, req)
			if err != nil {
				return err
			}
			return ctx.Result(200, out)
		})
	}
}

// toHTTPRouterPath converts google.api.http path parameter syntax {param} to httprouter's :param.
func toHTTPRouterPath(p string) string {
	var b strings.Builder
	b.Grow(len(p))
	for i := 0; i < len(p); i++ {
		if p[i] == '{' {
			b.WriteByte(':')
			i++
			for i < len(p) && p[i] != '}' {
				b.WriteByte(p[i])
				i++
			}
		} else {
			b.WriteByte(p[i])
		}
	}
	return b.String()
}

// extractPathParams extracts path parameter names from HTTPPath.
// e.g. "/api/v1/hello/{name}" → ["name"]
func extractPathParams(path string) []string {
	var params []string
	for i := 0; i < len(path); i++ {
		if path[i] == '{' {
			start := i + 1
			for i++; i < len(path) && path[i] != '}'; i++ {
			}
			if start < i {
				params = append(params, path[start:i])
			}
		}
	}
	return params
}

// injectPathParams injects path parameters into the request struct via reflection.
// proto field "name" maps to exported Go field "Name", "user_id" → "UserId".
func injectPathParams(req interface{}, paramNames []string, ctx Context) {
	if len(paramNames) == 0 {
		return
	}
	v := reflect.ValueOf(req)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return
	}
	s := v.Elem()
	t := s.Type()

	for _, name := range paramNames {
		// path value from httprouter
		pv := ctx.PathValue(name)
		if pv == "" {
			continue
		}
		// Try direct match on exported field name (name → Name)
		fieldName := snakeToPascal(name)
		f := s.FieldByName(fieldName)
		if !f.IsValid() || !f.CanSet() {
			// Fallback: try proto json tag match
			for i := range t.NumField() {
				if tag := t.Field(i).Tag.Get("json"); tag != "" && strings.Split(tag, ",")[0] == name {
					f = s.Field(i)
					break
				}
			}
			if !f.IsValid() || !f.CanSet() {
				continue
			}
		}
		if f.Kind() == reflect.String {
			f.SetString(pv)
		}
	}
}

// snakeToPascal converts snake_case to PascalCase.
// e.g. "user_id" → "UserId", "name" → "Name"
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// buildFastHandler creates a closure at registration time for zero []reflect.Value heap allocation at request time.
func buildFastHandler(method reflect.Value) transport.Handler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		var args [2]reflect.Value
		args[0] = reflect.ValueOf(ctx)
		args[1] = reflect.ValueOf(req)
		results := method.Call(args[:])

		var err error
		if len(results) > 1 && !results[1].IsNil() {
			err = results[1].Interface().(error)
		}
		return results[0].Interface(), err
	}
}

// RegisterHandler is a generic registration for a single method with zero reflection at request time.
//
//	http.RegisterHandler(s, "helloworld.Greeter/SayHello", srv.SayHello)
func RegisterHandler[Req, Res any](
	s *Server,
	operation string,
	fn func(context.Context, *Req) (*Res, error),
) {
	path := "/" + operation

	// ★ Pre-build middleware chain
	matched := s.mwManager.Match(path)
	wrapped := func(ctx context.Context, req interface{}) (interface{}, error) {
		return fn(ctx, req.(*Req))
	}
	prebuilt := middleware.Chain(matched...)(wrapped)

	s.Handle("POST", path, func(ctx Context) error {
		var req Req

		if err := ctx.Bind(&req); err != nil {
			return err
		}

		out, err := prebuilt(ctx, &req)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}

// RegisterUnary is a generic route registration function called by generated code.
// Middleware chain is pre-built at registration, zero allocation and zero reflection at request time.
// The dec function decodes the request from the HTTP context (body, path params, etc.).
func RegisterUnary[Req, Res any](
	s *Server,
	method, path string,
	fn func(context.Context, *Req) (*Res, error),
	dec func(Context, *Req) error,
) {
	wrapped := func(ctx context.Context, req interface{}) (interface{}, error) {
		return fn(ctx, req.(*Req))
	}
	matched := s.mwManager.Match(path)
	prebuilt := middleware.Chain(matched...)(wrapped)

	s.Handle(method, path, func(ctx Context) error {
		var req Req
		if err := dec(ctx, &req); err != nil {
			return err
		}
		out, err := prebuilt(ctx, &req)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}
