package rpcx

import (
	"context"
	"net"

	rpcxServer "github.com/smallnest/rpcx/server"

	"github.com/neo532/gofr/middleware"
)

// ServerOption configures the rpcx server.
type ServerOption func(*Server)

// Address sets the listen address.
func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

// Network sets the network type ("tcp", "udp", etc.). Default "tcp".
func Network(n string) ServerOption {
	return func(s *Server) { s.network = n }
}

// Middleware registers global middlewares applied to all methods.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

// RpcxOptions passes raw rpcx server.OptionFn to the underlying rpcx server.
// e.g. rpcx.RpcxOptions(rpcxServer.WithReadTimeout(10*time.Second))
func RpcxOptions(opts ...rpcxServer.OptionFn) ServerOption {
	return func(s *Server) { s.rpcxOpts = append(s.rpcxOpts, opts...) }
}

// Server wraps rpcx server.Server and implements transport.Server.
type Server struct {
	*rpcxServer.Server
	network   string
	address   string
	lis       net.Listener
	mwManager *MiddlewareManager
	rpcxOpts  []rpcxServer.OptionFn
}

// Addr returns the actual listening address, available after Start.
func (s *Server) Addr() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}
	return s.address
}

// NewServer creates an rpcx server with middleware support.
// HTTP and JSON gateways are disabled — rpcx runs as a pure RPC transport.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		network:   "tcp",
		mwManager: newMiddlewareManager(),
	}
	for _, o := range opts {
		o(s)
	}
	s.Server = rpcxServer.NewServer(s.rpcxOpts...)
	s.Server.DisableHTTPGateway = true
	s.Server.DisableJSONRPC = true
	s.Plugins.Add(&middlewarePlugin{mwManager: s.mwManager})
	return s
}

// Use registers global middlewares.
func (s *Server) Use(m ...middleware.Middleware) {
	s.mwManager.Use(m...)
}

// UseWith registers middlewares scoped to a specific method path.
func (s *Server) UseWith(method string, m ...middleware.Middleware) {
	s.mwManager.UseWith(method, m...)
}

// Start implements transport.Server.
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen(s.network, s.address)
	if err != nil {
		return err
	}
	s.lis = lis

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	go s.ServeListener(s.network, lis)
	return nil
}

// Stop implements transport.Server.
func (s *Server) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}

// RegisterServiceWith registers a service with per-method middleware prebuilding.
// Compatible with generated code for zero-reflection registration.
func RegisterServiceWith(s *Server, serviceName string, svr any) {
	if err := s.RegisterName(serviceName, svr, ""); err != nil {
		panic("rpcx: RegisterName(" + serviceName + "): " + err.Error())
	}
}
