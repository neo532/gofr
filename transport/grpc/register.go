package grpc

import (
	"context"
	"net"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// ServerOption configures the gRPC server.
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

// GrpcOptions passes raw grpc.ServerOption to the underlying grpc.Server.
func GrpcOptions(opts ...grpc.ServerOption) ServerOption {
	return func(s *Server) { s.grpcOpts = append(s.grpcOpts, opts...) }
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

// Server wraps grpc.Server and implements transport.Server with middleware.
type Server struct {
	*grpc.Server
	address        string
	endpointHost   string
	lis            net.Listener
	ready          chan struct{}
	mwManager      *MiddlewareManager
	grpcOpts       []grpc.ServerOption
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
	return transport.EndpointURL("grpc", s.address, s.endpointHost)
}

// NewServer creates a gRPC server with gofr options.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		mwManager: newMiddlewareManager(),
		ready:     make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	s.grpcOpts = append([]grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryServerInterceptor(s)),
	}, s.grpcOpts...)
	s.Server = grpc.NewServer(s.grpcOpts...)
	return s
}

// Use registers global middlewares applied to all methods.
func (s *Server) Use(m ...middleware.Middleware) {
	s.mwManager.Use(m...)
}

// UseWith registers middlewares scoped to a specific method path (e.g. "/helloworld.Greeter/SayHello").
func (s *Server) UseWith(method string, m ...middleware.Middleware) {
	s.mwManager.UseWith(method, m...)
}

// unaryServerInterceptor wraps the context with a Transporter carrying request/reply metadata.
func unaryServerInterceptor(s *Server) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		incomingMD, _ := metadata.FromIncomingContext(ctx)
		replyMD := make(metadata.MD)

		tr := &Transport{
			endpoint:       s.address,
			operation:      info.FullMethod,
			reqHeader:      headerCarrier(incomingMD),
			replyHeader:    headerCarrier(replyMD),
			trustedProxies: s.trustedProxies,
			app:            s.app,
		}
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			tr.peer = p.Addr.String()
		}

		// Pre-register reply headers so modifications via Tr.ReplyHeader() are
		// automatically sent with the response.
		_ = grpc.SetHeader(ctx, replyMD)

		ctx = transport.NewServerContext(ctx, tr)
		return handler(ctx, req)
	}
}

// PrebuildHandler pre-computes middleware chain for a gRPC method.
func (s *Server) PrebuildHandler(fullMethod string, fn transport.Handler) transport.Handler {
	matched := s.mwManager.Match(fullMethod)
	return middleware.Chain(matched...)(fn)
}

// Start implements transport.Server.
func (s *Server) Start(ctx context.Context) error {
	if s.lis == nil {
		var err error
		s.lis, err = net.Listen("tcp", s.address)
		if err != nil {
			return err
		}
	}
	close(s.ready)
	transport.LogListen(ctx, s.app, s.lis, transport.KindGRPC)

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	return s.Server.Serve(s.lis)
}

// Stop implements transport.Server with context deadline.
func (s *Server) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.Server.Stop()
		return ctx.Err()
	}
}

// UnaryHandler is a direct type-safe handler for a single gRPC method.
// Used by generated code for zero-reflection registration.
type UnaryHandler func(ctx context.Context, req any) (any, error)

// RegisterServiceWith registers a multi-method gRPC service with direct handlers.
// Middleware is applied per method via PrebuildHandler.
func RegisterServiceWith(s *Server, serviceName string, svr any, methods []struct {
	Name    string
	NewReq  func() any
	Handler UnaryHandler
}) {
	desc := &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*any)(nil),
	}
	for _, m := range methods {
		md := m
		fullMethod := "/" + serviceName + "/" + md.Name
		wrapped := s.PrebuildHandler(fullMethod, transport.Handler(md.Handler))

		desc.Methods = append(desc.Methods, grpc.MethodDesc{
			MethodName: md.Name,
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				req := md.NewReq()
				if err := dec(req); err != nil {
					return nil, err
				}

				if interceptor != nil {
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: fullMethod,
					}
					return interceptor(ctx, req, info, func(ctx context.Context, req any) (any, error) {
						return wrapped(ctx, req)
					})
				}
				return wrapped(ctx, req)
			},
		})
	}
	s.Server.RegisterService(desc, svr)
}

