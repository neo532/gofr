package script

import (
	"context"
	"flag"
	"reflect"
	"strings"
	"syscall"

	"github.com/neo532/gokit/errorx"
)

// Func is the signature for a script handler.
type Func func(context.Context, ...string) error

// Server runs a script function on Start and signals shutdown on completion.
type Server struct {
	router map[string]Func
}

// New creates a Server with the given route map.
func New(routes map[string]Func) *Server {
	return &Server{router: routes}
}

// Discover reflects on obj for methods matching
// func(context.Context, ...string) error and returns a route map.
// Route keys are lowercase "structName.methodName".
func Discover(obj any) map[string]Func {
	routes := make(map[string]Func)

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() != reflect.Pointer {
		return routes
	}

	structName := strings.ToLower(t.Elem().Name())
	structName = strings.TrimSuffix(structName, "script")
	if structName == "" {
		structName = strings.ToLower(t.Elem().Name())
	}

	for i := range t.NumMethod() {
		m := t.Method(i)
		if !matchFunc(m) {
			continue
		}

		name := structName + "." + strings.ToLower(m.Name)
		routes[name] = func(c context.Context, args ...string) error {
			in := make([]reflect.Value, 0, len(args)+1)
			in = append(in, reflect.ValueOf(c))
			for _, a := range args {
				in = append(in, reflect.ValueOf(a))
			}
			out := v.Method(m.Index).Call(in)
			if len(out) > 0 && !out[0].IsNil() {
				return out[0].Interface().(error)
			}
			return nil
		}
	}

	return routes
}

func matchFunc(m reflect.Method) bool {
	if m.Type.NumIn() != 3 || !m.Type.IsVariadic() {
		return false
	}
	return m.Type.In(1) == reflect.TypeFor[context.Context]() &&
		m.Type.In(2) == reflect.TypeFor[[]string]() &&
		m.Type.NumOut() == 1 &&
		m.Type.Out(0) == reflect.TypeFor[error]()
}

// Start looks up the script by the first CLI argument, executes it,
// then signals the process to shut down gracefully.
func (s *Server) Start(c context.Context) (err error) {
	args := flag.Args()
	if len(args) < 1 {
		return errorx.New("script name required")
	}
	fn, ok := s.router[args[0]]
	if !ok {
		keys := make([]string, 0, len(s.router))
		for k := range s.router {
			keys = append(keys, k)
		}
		return errorx.New("unknown script %q, available: %v", args[0], keys)
	}
	if err = fn(c, args[1:]...); err != nil {
		return
	}
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	return
}

// Stop is a no-op; shutdown is handled by Start signalling SIGINT.
func (s *Server) Stop(context.Context) error {
	return nil
}
