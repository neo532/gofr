package http

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

var wrapperPool = sync.Pool{
	New: func() any { return &wrapper{} },
}

var transportPool = sync.Pool{
	New: func() any { return &Transport{} },
}

// ServerOption configures the HTTP server.
type ServerOption func(*Server)

func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

func Timeout(d time.Duration) ServerOption {
	return func(s *Server) { s.timeout = d }
}

func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) { s.tlsConf = c }
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

// Server is an HTTP server wrapper based on httprouter.
type Server struct {
	router    *httprouter.Router
	srv       atomic.Value // *http.Server, set in Start()
	address   string
	timeout   time.Duration
	tlsConf   *tls.Config
	lis       net.Listener
	decBody   DecodeRequestFunc
	enc       EncodeResponseFunc
	ene       EncodeErrorFunc
	mwManager *MiddlewareManager
}

// Addr returns the actual listening address, available after Start.
func (s *Server) Addr() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}
	return s.address
}

// NewServer creates an HTTP server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		address:   ":0",
		timeout:   30 * time.Second,
		router:    httprouter.New(),
		decBody:   DefaultRequestDecoder,
		enc:       DefaultResponseEncoder,
		ene:       DefaultErrorEncoder,
		mwManager: newMiddlewareManager(),
	}
	s.router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v any) {
		http.Error(w, "panic", http.StatusInternalServerError)
	}
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

// Handle registers a handler function with method and httprouter path.
func (s *Server) Handle(method, path string, handler func(Context) error) {
	s.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tr := transportPool.Get().(*Transport)
		tr.operation = path
		tr.reqHeader = headerCarrier(r.Header)
		tr.replyHeader = headerCarrier(w.Header())
		r = r.WithContext(transport.NewServerContext(r.Context(), tr))

		ctxw := wrapperPool.Get().(*wrapper)
		ctxw.req = r
		ctxw.res = w
		ctxw.srv = s
		ctxw.codec = s.decBody
		ctxw.params = ps

		err := handler(ctxw)
		if err != nil {
			s.ene(w, r, err)
		}

		ctxw.req = nil
		ctxw.res = nil
		ctxw.srv = nil
		ctxw.params = nil
		wrapperPool.Put(ctxw)

		tr.operation = ""
		tr.reqHeader = nil
		tr.replyHeader = nil
		transportPool.Put(tr)
	})
}

// HandleHandler registers a standard http.Handler.
func (s *Server) HandleHandler(method, path string, handler http.Handler) {
	s.router.Handler(method, path, handler)
}

// GET registers a GET handler.
func (s *Server) GET(path string, handler func(Context) error) {
	s.Handle("GET", path, handler)
}

// POST registers a POST handler.
func (s *Server) POST(path string, handler func(Context) error) {
	s.Handle("POST", path, handler)
}

// PUT registers a PUT handler.
func (s *Server) PUT(path string, handler func(Context) error) {
	s.Handle("PUT", path, handler)
}

// DELETE registers a DELETE handler.
func (s *Server) DELETE(path string, handler func(Context) error) {
	s.Handle("DELETE", path, handler)
}

// PATCH registers a PATCH handler.
func (s *Server) PATCH(path string, handler func(Context) error) {
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
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.lis = lis

	srv := &http.Server{
		Addr:        s.address,
		Handler:     s.router,
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
