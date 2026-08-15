package rpcx

import (
	"context"
	"net"
	"net/url"
	"sync"

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

// EndpointHost sets the advertised host for registration (the IP clients use
// to reach this instance). When unset, Endpoint falls back to the local IP.
func EndpointHost(host string) ServerOption {
	return func(s *Server) { s.endpointHost = host }
}

// Network sets the network type ("tcp", "udp", etc.). Default "tcp".
func Network(n string) ServerOption {
	return func(s *Server) { s.network = n }
}

// Middleware registers global middlewares applied to all methods.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

// TrustedProxies configures the reverse-proxy address ranges whose
// X-Forwarded-For / X-Real-IP metadata ClientIP will trust (see
// transport.ClientIP). When unset, metadata from any peer is trusted.
func TrustedProxies(cidrs ...string) ServerOption {
	return func(s *Server) {
		s.trustedProxies = append(s.trustedProxies, transport.ParseTrustedProxies(cidrs...)...)
	}
}

// RpcxOptions passes raw rpcx server.OptionFn to the underlying rpcx server.
// e.g. rpcx.RpcxOptions(rpcxServer.WithReadTimeout(10*time.Second))
func RpcxOptions(opts ...rpcxServer.OptionFn) ServerOption {
	return func(s *Server) { s.rpcxOpts = append(s.rpcxOpts, opts...) }
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

// Server wraps rpcx server.Server and implements transport.Server.
type Server struct {
	*rpcxServer.Server
	network        string
	address        string
	endpointHost   string
	lis            net.Listener
	ready          chan struct{}
	mwManager      *MiddlewareManager
	trustedProxies []*net.IPNet
	rpcxOpts       []rpcxServer.OptionFn
	app            transport.App

	svcMu    sync.RWMutex
	svcNames []string
}

func (s *Server) addServiceName(name string) {
	s.svcMu.Lock()
	s.svcNames = append(s.svcNames, name)
	s.svcMu.Unlock()
}

func (s *Server) serviceNames() []string {
	s.svcMu.RLock()
	defer s.svcMu.RUnlock()
	out := make([]string, len(s.svcNames))
	copy(out, s.svcNames)
	return out
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
	return transport.EndpointURL("rpcx", s.address, s.endpointHost)
}

// NewServer creates an rpcx server with middleware support.
// HTTP and JSON gateways are disabled — rpcx runs as a pure RPC transport.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		network:   "tcp",
		mwManager: newMiddlewareManager(),
		ready:     make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	s.Server = rpcxServer.NewServer(s.rpcxOpts...)
	s.Server.DisableHTTPGateway = true
	s.Server.DisableJSONRPC = true
	s.Plugins.Add(&middlewarePlugin{mwManager: s.mwManager, trustedProxies: s.trustedProxies, srv: s})
	if err := s.Server.RegisterName(MetadataApiName, newMetadataApi(s.serviceNames), ""); err != nil {
		panic("rpcx: RegisterName(" + MetadataApiName + "): " + err.Error())
	}
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
	if s.lis == nil {
		var err error
		s.lis, err = net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
	}
	close(s.ready)
	transport.LogListen(ctx, s.app, s.lis, transport.KindRPCX)

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	go s.ServeListener(s.network, s.lis)
	return nil
}

// Stop implements transport.Server.
func (s *Server) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}

// RegisterServiceWith registers a service with per-method middleware prebuilding.
// Compatible with generated code for zero-reflection registration. The name is
// recorded so the MetadataApi discovery service can dump its descriptors.
func RegisterServiceWith(s *Server, serviceName string, svr any) {
	if err := s.RegisterName(serviceName, svr, ""); err != nil {
		panic("rpcx: RegisterName(" + serviceName + "): " + err.Error())
	}
	s.addServiceName(serviceName)
}
