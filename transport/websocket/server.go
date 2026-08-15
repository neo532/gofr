// Package websocket provides a WebSocket server whose routes come from the
// same google.api.http annotations as the other gofr protocols.
//
// A proto method is registered under the HTTP verb and path of its annotation:
// { put: "/user/{userId}" } registers PUT on "/user/:userId". Two methods may
// share a path when their verbs differ; a duplicate (method, path) is rejected.
//
// A handshake selects the method in one of two ways:
//
//   - Native request line: "PUT /user/1 HTTP/1.1" reaches the put: method.
//   - "METHOD." path prefix: browsers always handshake with GET, so the verb
//     may ride in the path instead, e.g. "GET /PUT./user/1 HTTP/1.1" is
//     rewritten to PUT on "/user/1". The dot is the required delimiter — a path
//     that merely begins with a verb segment (e.g. "/PUT/user/1") is a real
//     annotated path and is never misrouted. Only uppercase verbs are treated
//     as selectors, so a path like "/post/123" is unaffected.
//
// A bare handshake with neither indicator defaults to GET.
package websocket

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// BinaryMessage is a sent for WebSocket binary frames (matches gorilla/websocket.BinaryMessage).
const BinaryMessage = 2

// Conn is a WebSocket connection (aliased from gorilla/websocket).
type Conn = websocket.Conn

// WsHandler handles an upgraded WebSocket connection.
type WsHandler func(ctx context.Context, conn *websocket.Conn) error

// ServerOption configures the WebSocket server.
type ServerOption func(*Server)

// Address sets the listen address.
func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

// EndpointHost sets the advertised host for registration (the IP clients use
// to reach this instance). When unset, Endpoint falls back to the local IP.
func EndpointHost(host string) ServerOption {
	return func(s *Server) { s.endpointHost = host }
}

// Middleware registers global middlewares applied to all WebSocket endpoints.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

// TrustedProxies configures the reverse-proxy address ranges whose
// X-Forwarded-For / X-Real-IP headers ClientIP will trust (see
// transport.ClientIP). When unset, headers from any peer are trusted.
func TrustedProxies(cidrs ...string) ServerOption {
	return func(s *Server) {
		s.trustedProxies = append(s.trustedProxies, transport.ParseTrustedProxies(cidrs...)...)
	}
}

// Timeout sets the read timeout on the underlying HTTP server.
func Timeout(d time.Duration) ServerOption {
	return func(s *Server) { s.timeout = d }
}

// WithListener injects an external listener (used by graceful restart).
func WithListener(lis net.Listener) ServerOption {
	return func(s *Server) { s.lis = lis }
}

// SetListener implements transport.ListenerServer for external listener injection.
func (s *Server) SetListener(lis net.Listener) { s.lis = lis }

// App injects the App so each request's Transporter can reach shared
// application resources (logger, etc.). Called by gofr.App.Run.
func (s *Server) App(a transport.App) { s.app = a }

// wsRoute pairs a registered HTTP verb with its path and handler. The verb
// comes from the google.api.http binding; a WebSocket handshake selects the
// route with its request-line method (e.g. "PUT /user/163 HTTP/1.1"), exactly
// as native HTTP does.
type wsRoute struct {
	method  string
	path    string
	handler WsHandler
}

// Server is a standalone WebSocket server implementing transport.Server.
type Server struct {
	address        string
	endpointHost   string
	lis            net.Listener
	ready          chan struct{}
	routes         []wsRoute
	router         *httprouter.Router
	upgrader       websocket.Upgrader
	mwManager      *MiddlewareManager
	httpSrv        *http.Server
	timeout        time.Duration
	trustedProxies []*net.IPNet
	app            transport.App
}

// NewServer creates a WebSocket server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		upgrader:  websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		mwManager: newMiddlewareManager(),
		httpSrv:   &http.Server{},
		ready:     make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Addr returns the actual listening address, available after Start.
func (s *Server) Addr() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}
	return s.address
}

// Ready returns a channel closed once the listener is bound.
func (s *Server) Ready() <-chan struct{} { return s.ready }

// Endpoint returns the advertised endpoint for service registration.
func (s *Server) Endpoint() (*url.URL, error) {
	return transport.EndpointURL("ws", s.address, s.endpointHost)
}

// Handle registers a WebSocket handler for a proto method. method is the
// google.api.http verb; a client selects it either with the request-line method
// of the handshake ("PUT /user/163 HTTP/1.1" reaches the put: route), or — for
// browsers, whose handshake is always GET — with the verb as a "METHOD." path
// prefix ("GET /PUT./user/163 HTTP/1.1", see splitMethodPath). A bare handshake
// defaults to GET, so "GET /user/1" reaches the get: route. Two methods may
// share a path when their verbs differ; a duplicate (method, path) is rejected.
// The path may use :param segments (e.g. "/room/:id"). Registration must happen
// before the server starts serving.
func (s *Server) Handle(method, path string, handler WsHandler) {
	if s.router != nil {
		panic("websocket: cannot register routes after the server has started")
	}
	s.routes = append(s.routes, wsRoute{method: method, path: path, handler: handler})
}

// buildRouter compiles the registered routes into an httprouter. Each route is
// registered under its native HTTP verb, so the request line selects the proto
// method exactly as in native HTTP: "GET /user/1 HTTP/1.1" reaches the get:
// route, "PUT /user/1 HTTP/1.1" the put: route.
func (s *Server) buildRouter() {
	r := httprouter.New()
	for _, rt := range s.routes {
		method, path, handler := rt.method, rt.path, rt.handler
		r.Handle(method, path, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
			s.serveWS(w, r, handler)
		})
	}
	s.router = r
}

// Use registers global middlewares.
func (s *Server) Use(m ...middleware.Middleware) {
	s.mwManager.Use(m...)
}

// UseWith registers middlewares scoped to a specific path.
func (s *Server) UseWith(path string, m ...middleware.Middleware) {
	s.mwManager.UseWith(path, m...)
}

// Start implements transport.Server.
func (s *Server) Start(ctx context.Context) error {
	if s.router == nil {
		s.buildRouter()
	}
	if s.lis == nil {
		var err error
		s.lis, err = net.Listen("tcp", s.address)
		if err != nil {
			return err
		}
	}
	close(s.ready)
	transport.LogListen(ctx, s.app, s.lis, transport.KindWebSocket)

	s.httpSrv.Handler = http.HandlerFunc(s.serveHTTP)
	s.httpSrv.ReadTimeout = s.timeout

	go func() {
		<-ctx.Done()
		s.httpSrv.Close()
	}()

	return s.httpSrv.Serve(s.lis)
}

// Stop implements transport.Server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// methodTokens are the google.api.http verbs a handshake may select via a
// "METHOD." path prefix. Only the uppercase form, followed by a dot, is treated
// as a selector: "/PUT./user/1" selects PUT on "/user/1", while a path that
// merely begins with a verb segment such as "/PUT/user/1" or "/post/123" is
// left untouched, so a legitimately annotated path is never misrouted.
var methodTokens = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

// splitMethodPath detects an optional leading "METHOD." token in a WebSocket
// handshake path, so a browser's GET-only handshake can still select a non-GET
// proto method: "/PUT./user/1" resolves to method PUT and path "/user/1". The
// dot is the required delimiter — a plain "/PUT/user/1" is not rewritten, since
// it may be a real annotated path. A path with no leading method token returns
// ok=false and is routed as-is, which defaults the method to GET.
func splitMethodPath(p string) (method, path string, ok bool) {
	q := strings.TrimPrefix(p, "/")
	for _, m := range methodTokens {
		if !strings.HasPrefix(q, m) {
			continue
		}
		rest := q[len(m):]
		if !strings.HasPrefix(rest, ".") {
			continue
		}
		return m, "/" + strings.TrimLeft(rest[1:], "/"), true
	}
	return "", "", false
}

// serveHTTP routes the request to its registered handler. A leading "METHOD."
// token in the path (e.g. "/PUT./user/1") is rewritten to the native method+path,
// letting a browser's GET handshake reach any verb. Paths with :param segments
// are matched by httprouter; unmatched paths fall through to a 404.
func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if m, p, ok := splitMethodPath(r.URL.Path); ok {
		r.Method = m
		r.URL.Path = p
	}
	s.router.ServeHTTP(w, r)
}

// serveWS runs the middleware chain, upgrades the connection and hands it to
// the handler. It is invoked from the httprouter route closure.
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request, handler WsHandler) {
	// Set up transport context
	tr := &wsTransport{
		endpoint:       r.Host,
		operation:      r.URL.Path,
		reqHeader:      headerCarrier(r.Header),
		peer:           r.RemoteAddr,
		trustedProxies: s.trustedProxies,
		app:            s.app,
	}
	ctx := transport.NewServerContext(r.Context(), tr)

	// Run middleware chain before upgrade. The post-middleware context is
	// captured so middleware-installed values (e.g. a trace span) reach the
	// handler.
	matched := s.mwManager.Match(r.URL.Path)
	if len(matched) > 0 {
		chain := middleware.Chain(matched...)
		var ctxIn context.Context
		_, err := chain(func(ctx context.Context, req any) (any, error) {
			ctxIn = ctx
			return nil, nil
		})(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if ctxIn != nil {
			ctx = ctxIn
		}
	}

	// Upgrade to WebSocket. httprouter already routed this request by its
	// method (request line, or a "METHOD." path prefix like "/PUT./user/1"
	// rewritten in serveHTTP), so non-GET routes reach the right handler.
	// gorilla's upgrader
	// rejects non-GET methods as a protocol-format check (RFC 6455 clients
	// always send GET), so we hand it a GET-cloned copy; nothing downstream
	// reads the method.
	if r.Method != http.MethodGet {
		upgradeReq := r.Clone(r.Context())
		upgradeReq.Method = http.MethodGet
		r = upgradeReq
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	handler(ctx, conn)
}
