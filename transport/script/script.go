package script

import (
	"context"
	"flag"
	"net"
	"reflect"
	"strings"
	"syscall"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gokit/errorx"
)

// Func is the signature for a script handler.
type Func func(context.Context, ...string) error

// ServerOption configures the script server.
type ServerOption func(*Server)

// Middleware registers global middlewares applied to all scripts.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

// Server runs a script function on Start and signals shutdown on completion.
type Server struct {
	router    map[string]Func
	mwManager *MiddlewareManager
	app       transport.App
}

// App injects the application's shared resources (logger, etc.) so scripts
// can reach them via transport.FromServerContext(ctx).App(). Called by gofr.App.Run.
func (s *Server) App(a transport.App) { s.app = a }

// New creates a Server with the given route map.
func New(routes map[string]Func, opts ...ServerOption) *Server {
	s := &Server{
		router:    routes,
		mwManager: newMiddlewareManager(),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Use registers global middlewares.
func (s *Server) Use(m ...middleware.Middleware) {
	s.mwManager.Use(m...)
}

// UseWith registers middlewares scoped to a specific script name.
func (s *Server) UseWith(name string, m ...middleware.Middleware) {
	s.mwManager.UseWith(name, m...)
}

// MiddlewareManager manages middleware by script name.
type MiddlewareManager struct {
	global []middleware.Middleware
	opMap  map[string][]middleware.Middleware // script name → middlewares
}

func newMiddlewareManager() *MiddlewareManager {
	return &MiddlewareManager{opMap: make(map[string][]middleware.Middleware)}
}

// Use adds global middleware.
func (m *MiddlewareManager) Use(mw ...middleware.Middleware) {
	m.global = append(m.global, mw...)
}

// UseWith adds middleware scoped to a specific script name.
func (m *MiddlewareManager) UseWith(name string, mw ...middleware.Middleware) {
	m.opMap[name] = append(m.opMap[name], mw...)
}

// Match returns all middlewares matching the script (global + specific).
func (m *MiddlewareManager) Match(name string) []middleware.Middleware {
	total := len(m.global) + len(m.opMap[name])
	if total == 0 {
		return nil
	}
	ms := make([]middleware.Middleware, 0, total)
	ms = append(ms, m.global...)
	ms = append(ms, m.opMap[name]...)
	return ms
}

// scriptTransport carries the command line and App into a script's context so
// middlewares can log the executed command and scripts can query shared
// application resources like the logger.
type scriptTransport struct {
	app     transport.App
	command string
	args    []string
}

func (t *scriptTransport) Kind() transport.Kind           { return transport.KindScript }
func (t *scriptTransport) Endpoint() string                { return "" }
func (t *scriptTransport) Operation() string               { return "" }
func (t *scriptTransport) RequestHeader() transport.Header { return nil }
func (t *scriptTransport) ReplyHeader() transport.Header   { return nil }
func (t *scriptTransport) App() transport.App              { return t.app }

// Command returns the executed command name followed by its arguments.
func (t *scriptTransport) Command() []string {
	return append([]string{t.command}, t.args...)
}

// ClientIP returns the IPv4 of the eth0 interface, falling back to the first
// non-loopback IPv4 address when eth0 is absent (macOS, some cloud images).
func (t *scriptTransport) ClientIP() string {
	if ip := interfaceIPv4("eth0"); ip != "" {
		return ip
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok {
			if ip4 := ipNet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
				return ip4.String()
			}
		}
	}
	return ""
}

func interfaceIPv4(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok {
			if ip4 := ipNet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
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
	if err = s.run(c, args); err != nil {
		return
	}
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	return
}

// run executes the script named args[0] with the remaining arguments, wrapped
// in the middleware chain matched for that script. It is split from Start so
// tests can drive scripts without touching the global flag state.
func (s *Server) run(c context.Context, args []string) (err error) {
	fn, ok := s.router[args[0]]
	if !ok {
		keys := make([]string, 0, len(s.router))
		for k := range s.router {
			keys = append(keys, k)
		}
		return errorx.New("unknown script %q, available: %v", args[0], keys)
	}
	c = transport.NewServerContext(c, &scriptTransport{
		app:     s.app,
		command: args[0],
		args:    args[1:],
	})
	if mws := s.mwManager.Match(args[0]); len(mws) > 0 {
		chain := middleware.Chain(mws...)
		_, err = chain(func(ctx context.Context, req any) (any, error) {
			return nil, fn(ctx, args[1:]...)
		})(c, nil)
		return
	}
	return fn(c, args[1:]...)
}

// Stop is a no-op; shutdown is handled by Start signalling SIGINT.
func (s *Server) Stop(context.Context) error {
	return nil
}
