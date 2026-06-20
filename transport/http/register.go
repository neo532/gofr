package http

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// RegisterService registers all methods of a service via ServiceDesc.
// MethodByName runs at registration time (startup), leaving only reflect.Call per request.
func RegisterService(s *Server, desc *transport.ServiceDesc, svr interface{}) {
	val := reflect.ValueOf(svr)

	for _, m := range desc.Methods {
		method := val.MethodByName(m.Name)
		if !method.IsValid() {
			panic(fmt.Sprintf("gofr: service %q has no method %q", desc.Name, m.Name))
		}

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
		paramNames := extractPathParams(m.HTTPPath)

		fastHandler := buildFastHandler(method)

		matched := s.mwManager.Match(operation)
		prebuilt := middleware.Chain(matched...)(fastHandler)

		s.Handle(httpMethod, path, func(ctx Context) error {
			req := m.NewRequest()

			if err := ctx.Bind(req); err != nil {
				return err
			}

			injectPathParams(req, paramNames, ctx)

			out, err := prebuilt(ctx, req)
			if err != nil {
				return err
			}
			return ctx.Result(200, out)
		})
	}
}

// RegisterHandler is a generic registration for a single method with zero reflection at request time.
func RegisterHandler[Req, Res any](
	s *Server,
	operation string,
	fn func(context.Context, *Req) (*Res, error),
) {
	path := "/" + operation

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

// -- helpers for RegisterService (no per-request reflection beyond reflect.Call) --

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
		pv := ctx.PathValue(name)
		if pv == "" {
			continue
		}
		fieldName := snakeToPascal(name)
		f := s.FieldByName(fieldName)
		if !f.IsValid() || !f.CanSet() {
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
		switch f.Kind() {
		case reflect.String:
			f.SetString(pv)
		case reflect.Int64, reflect.Int32, reflect.Int:
			n, _ := strconv.ParseInt(pv, 10, 64)
			f.SetInt(n)
		case reflect.Uint64, reflect.Uint32, reflect.Uint:
			n, _ := strconv.ParseUint(pv, 10, 64)
			f.SetUint(n)
		case reflect.Float64, reflect.Float32:
			n, _ := strconv.ParseFloat(pv, 64)
			f.SetFloat(n)
		case reflect.Bool:
			n, _ := strconv.ParseBool(pv)
			f.SetBool(n)
		}
	}
}

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// buildFastHandler pre-resolves reflect.Value at registration; only reflect.Call remains per request.
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
