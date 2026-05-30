package rpcx

import (
	"context"
	"net"

	rpcxServer "github.com/smallnest/rpcx/server"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
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

// Server wraps rpcx server.Server and implements transport.Server.
type Server struct {
	*rpcxServer.Server
	network   string
	address   string
	lis       net.Listener
	mwManager *MiddlewareManager
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
func NewServer(address string, opts ...ServerOption) *Server {
	s := &Server{
		Server:    rpcxServer.NewServer(),
		network:   "tcp",
		address:   address,
		mwManager: newMiddlewareManager(),
	}
	s.Server.DisableHTTPGateway = true
	s.Server.DisableJSONRPC = true
	for _, o := range opts {
		o(s)
	}
	// Attach middleware as an rpcx plugin
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
func RegisterServiceWith(s *Server, serviceName string, svr interface{}) {
	if err := s.RegisterName(serviceName, svr, ""); err != nil {
		panic("rpcx: RegisterName(" + serviceName + "): " + err.Error())
	}
}

// RegisterService registers a transport.ServiceDesc onto the rpcx server.
// Deprecated: use RegisterServiceWith and generated code.
func RegisterService(srv *Server, desc *transport.ServiceDesc, svr interface{}) {
	for _, m := range desc.Methods {
		fullMethod := "/" + desc.Name + "/" + m.Name
		_ = fullMethod
		srv.AddHandler(desc.Name, m.Name, func(ctx *rpcxServer.Context) error {
			req := m.NewRequest()
			if err := ctx.Bind(req); err != nil {
				return err
			}
			return nil
		})
	}
}
