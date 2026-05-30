package websocket

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// WsHandler handles an upgraded WebSocket connection.
type WsHandler func(ctx context.Context, conn *websocket.Conn) error

// ServerOption configures the WebSocket server.
type ServerOption func(*Server)

// Address sets the listen address.
func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

// Middleware registers global middlewares applied to all WebSocket endpoints.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

// Server is a standalone WebSocket server implementing transport.Server.
type Server struct {
	address   string
	lis       net.Listener
	mux       map[string]WsHandler
	mu        sync.RWMutex
	upgrader  websocket.Upgrader
	mwManager *MiddlewareManager
	httpSrv   *http.Server
}

// NewServer creates a WebSocket server.
func NewServer(address string, opts ...ServerOption) *Server {
	s := &Server{
		address:   address,
		mux:       make(map[string]WsHandler),
		upgrader:  websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		mwManager: newMiddlewareManager(),
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

// Handle registers a WebSocket handler at the given path.
func (s *Server) Handle(path string, handler WsHandler) {
	s.mu.Lock()
	s.mux[path] = handler
	s.mu.Unlock()
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
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.lis = lis

	s.httpSrv = &http.Server{
		Handler: http.HandlerFunc(s.serveHTTP),
	}

	go func() {
		<-ctx.Done()
		s.httpSrv.Close()
	}()

	return s.httpSrv.Serve(lis)
}

// Stop implements transport.Server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// serveHTTP handles the HTTP request, upgrades to WebSocket if path matches.
func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	handler, ok := s.mux[r.URL.Path]
	s.mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	// Set up transport context
	tr := &wsTransport{
		endpoint:  r.Host,
		operation: r.URL.Path,
		reqHeader: headerCarrier(r.Header),
	}
	ctx := transport.NewServerContext(r.Context(), tr)

	// Run middleware chain before upgrade
	matched := s.mwManager.Match(r.URL.Path)
	if len(matched) > 0 {
		chain := middleware.Chain(matched...)
		_, err := chain(func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, nil
		})(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	handler(ctx, conn)
}
