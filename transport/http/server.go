package http

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// ServerOption configures the HTTP server.
type ServerOption func(*Server)

func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

func Timeout(d time.Duration) ServerOption {
	return func(s *Server) { s.timeout = d }
}

// EndpointHost sets the advertised host for registration (the IP clients use
// to reach this instance). When unset, Endpoint falls back to the local IP.
func EndpointHost(host string) ServerOption {
	return func(s *Server) { s.endpointHost = host }
}

func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) { s.tlsConf = c }
}

func PanicHandler(fn func(w http.ResponseWriter, r *http.Request, v any)) ServerOption {
	return func(s *Server) { s.router.PanicHandler = fn }
}

func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

func RequestDecoder(dec DecodeRequestFunc) ServerOption {
	return func(s *Server) { s.decBody = dec }
}

func ResponseEncoder(enc EncodeResponseFunc) ServerOption {
	return func(s *Server) { s.enc = enc }
}

func ErrorEncoder(ene EncodeErrorFunc) ServerOption {
	return func(s *Server) { s.ene = ene }
}

// WithListener injects an external listener (used by graceful restart).
// When set, Start skips net.Listen and uses this listener directly.
func WithListener(lis net.Listener) ServerOption {
	return func(s *Server) { s.lis = lis }
}

// TrustedProxies configures the reverse-proxy address ranges whose
// X-Forwarded-For / X-Real-IP headers ClientIP will trust. Accepts CIDRs
// like "10.0.0.0/8" or single IPs like "127.0.0.1". A request whose direct
// peer is NOT listed always reports the direct RemoteAddr, so a client
// cannot spoof its real IP with forged headers. When unset, headers from any
// peer are trusted (see transport.ClientIP).
func TrustedProxies(cidrs ...string) ServerOption {
	return func(s *Server) {
		s.trustedProxies = append(s.trustedProxies, transport.ParseTrustedProxies(cidrs...)...)
	}
}

// SetListener implements transport.ListenerServer for external listener injection.
func (s *Server) SetListener(lis net.Listener) { s.lis = lis }

// App injects the App so each request's Transporter can reach shared
// application resources (logger, etc.). Called by gofr.App.Run.
func (s *Server) App(a transport.App) { s.app = a }

// Server is an HTTP server wrapper based on httprouter.
type Server struct {
	router         *httprouter.Router
	srv            atomic.Value // *http.Server, set in Start()
	address        string
	endpointHost   string
	timeout        time.Duration
	tlsConf        *tls.Config
	lis            net.Listener
	ready          chan struct{}
	decBody        DecodeRequestFunc
	enc            EncodeResponseFunc
	ene            EncodeErrorFunc
	mwManager      *MiddlewareManager
	trustedProxies []*net.IPNet
	app            transport.App
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
	return transport.EndpointURL("http", s.address, s.endpointHost)
}

// NewServer creates an HTTP server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		address:   ":0",
		timeout:   30 * time.Second,
		router:    httprouter.New(),
		ready:     make(chan struct{}),
		decBody:   DefaultRequestDecoder,
		enc:       DefaultResponseEncoder,
		ene:       DefaultErrorEncoder,
		mwManager: newMiddlewareManager(),
	}
	s.router.PanicHandler = s.DefaultPanicHandler
	for _, o := range opts {
		o(s)
	}
	return s
}

// Use registers global middlewares applied to all routes.
func (s *Server) Use(m ...middleware.Middleware) {
	s.mwManager.Use(m...)
}

// UseWith registers middlewares scoped to a specific operation path.
func (s *Server) UseWith(operation string, m ...middleware.Middleware) {
	s.mwManager.UseWith(operation, m...)
}

// Handle registers a raw handler with method and httprouter path. The handler
// receives the standard (w, r, ps) triple; path params are reachable via ps,
// and the Transporter injected by the top-level handler in Start is available
// from r.Context(). The route path is stamped onto the per-request Transport so
// middleware and logging observe the route template, not just the matched URL.
func (s *Server) Handle(method, path string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if tr, ok := r.Context().Value(transport.ServerTransportKey{}).(*Transport); ok {
			tr.operation = path
		}
		handler(w, r, ps)
	})
}

// HandleUnary registers a generated-style unary route. It decodes the request
// body via the server's request decoder, applies path-param bindings via dec,
// runs the pre-built middleware chain, then encodes the reply or error. dec may
// be nil when the route carries no path params. This is the zero-reflection
// path used by generated code. (Package-level because Go methods cannot declare
// their own type parameters.)
func HandleUnary[Req, Res any](
	s *Server,
	method, path string,
	fn func(context.Context, *Req) (*Res, error),
	dec func(httprouter.Params, *Req) error,
) {
	wrapped := func(ctx context.Context, req any) (any, error) { return fn(ctx, req.(*Req)) }
	prebuilt := s.PrebuildHandler(path, transport.Handler(wrapped))

	s.Handle(method, path, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		req := new(Req)
		if err := s.decBody(r, req); err != nil {
			s.ene(w, r, err)
			return
		}
		if dec != nil {
			if err := dec(ps, req); err != nil {
				s.ene(w, r, err)
				return
			}
		}
		out, err := prebuilt(r.Context(), req)
		if err != nil {
			s.ene(w, r, err)
			return
		}
		s.enc(w, r, out)
	})
}

// HandleHandler registers a standard http.Handler.
func (s *Server) HandleHandler(method, path string, handler http.Handler) {
	s.router.Handler(method, path, handler)
}

// GET registers a GET handler.
func (s *Server) GET(path string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.Handle("GET", path, handler)
}

// POST registers a POST handler.
func (s *Server) POST(path string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.Handle("POST", path, handler)
}

// PUT registers a PUT handler.
func (s *Server) PUT(path string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.Handle("PUT", path, handler)
}

// DELETE registers a DELETE handler.
func (s *Server) DELETE(path string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.Handle("DELETE", path, handler)
}

// PATCH registers a PATCH handler.
func (s *Server) PATCH(path string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.Handle("PATCH", path, handler)
}

// PrebuildHandler pre-computes middleware chain for an operation.
// Used by generated code for zero-reflection handler registration.
func (s *Server) PrebuildHandler(operation string, fn transport.Handler) transport.Handler {
	matched := s.mwManager.Match(operation)
	return middleware.Chain(matched...)(fn)
}

// Start implements transport.Server.
func (s *Server) Start(ctx context.Context) error {
	var err error
	if s.lis == nil {
		s.lis, err = net.Listen("tcp", s.address)
		if err != nil {
			return err
		}
	}
	close(s.ready)
	transport.LogListen(ctx, s.app, s.lis, transport.KindHTTP)

	// Each request gets a fresh Transport allocated and injected into its request
	// context via context.WithValue. The context chain is immutable and GC'd with
	// the request, so async goroutines that keep the context (e.g. gorm's
	// Rows.awaitDone) can call FromServerContext at any time and always see a
	// valid, fully-populated Transport — never a recycled one.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := &Transport{
			reqHeader:      headerCarrier(r.Header),
			replyHeader:    headerCarrier(w.Header()),
			peer:           r.RemoteAddr,
			trustedProxies: s.trustedProxies,
			app:            s.app,
			req:            r,
		}
		r = r.WithContext(transport.NewServerContext(r.Context(), tr))
		s.router.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:        s.address,
		Handler:     handler,
		TLSConfig:   s.tlsConf,
		ReadTimeout: s.timeout,
	}
	s.srv.Store(srv)

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	if s.tlsConf != nil {
		err = srv.ServeTLS(s.lis, "", "")
	} else {
		err = srv.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

// Stop implements transport.Server.
func (s *Server) Stop(ctx context.Context) error {
	if srv, ok := s.srv.Load().(*http.Server); ok {
		return srv.Shutdown(ctx)
	}
	return nil
}
