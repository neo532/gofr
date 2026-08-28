package grpc

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/middleware/manager"
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/transport/ip"
	"github.com/neo532/gofr/transport/route"
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

// WithMiddlewareManager injects an external composite middleware manager,
// letting business projects supply their own prefix/regex/exact matchers.
// Globals registered via the Middleware option are carried over, so option
// order does not matter.
func WithMiddlewareManager(m *manager.MiddlewareManager) ServerOption {
	return func(s *Server) {
		s.mwManager = m.Use(s.mwManager.Global()...)
	}
}

// TrustedProxies configures the reverse-proxy address ranges whose
// X-Forwarded-For / X-Real-IP metadata ClientIP will trust (see
// transport/ip.ClientIP). When unset, metadata from any peer is trusted.
func TrustedProxies(cidrs ...string) ServerOption {
	return func(s *Server) {
		s.trustedProxies = append(s.trustedProxies, ip.ParseTrustedProxies(cidrs...)...)
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
	mwManager      *manager.MiddlewareManager
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
	host, port, err := net.SplitHostPort(s.address)
	if err != nil {
		return nil, fmt.Errorf("endpoint grpc: %q: %w", s.address, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = s.endpointHost
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = ip.LocalIP()
	}
	return &url.URL{Scheme: "grpc", Host: net.JoinHostPort(host, port)}, nil
}

// NewServer creates a gRPC server with gofr options.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		mwManager: manager.NewMiddlewareManager(),
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

// opFor resolves the HTTP-bound operation for a gRPC full method, falling back
// to the full method name when the method has no google.api.http binding.
func opFor(fullMethod string) (transport.Operation, string) {
	if op, tmpl, ok := route.RouteOperation(fullMethod); ok {
		return op, tmpl
	}
	return transport.Operation{Operation: fullMethod}, ""
}

// unaryServerInterceptor wraps the context with a Transporter carrying request/reply metadata.
func unaryServerInterceptor(s *Server) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		incomingMD, _ := metadata.FromIncomingContext(ctx)
		replyMD := make(metadata.MD)

		op, tmpl := opFor(info.FullMethod)
		tr := &Transport{
			endpoint:       s.address,
			operation:      op.Operation,
			method:         op.Method,
			routeKey:       strings.TrimPrefix(info.FullMethod, "/"),
			pathTmpl:       tmpl,
			reqHeader:      headerCarrier(incomingMD),
			replyHeader:    headerCarrier(replyMD),
			trustedProxies: s.trustedProxies,
			app:            s.app,
		}
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			tr.peer = p.Addr.String()
		}
		tr.SetReq(req)

		// Pre-register reply headers so modifications via Tr.ReplyHeader() are
		// automatically sent with the response.
		_ = grpc.SetHeader(ctx, replyMD)

		ctx = transport.NewServerContext(ctx, tr)
		return handler(ctx, req)
	}
}

// PrebuildHandler pre-computes middleware chain for a gRPC method.
func (s *Server) PrebuildHandler(op transport.Operation, fn middleware.Handler) middleware.Handler {
	matched := s.mwManager.Match(op)
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
	if s.app != nil {
		s.app.Logger().Info(ctx, "listening on", transport.KindKey, transport.KindGRPC, "addr", s.lis.Addr().String())
	}

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

// ServiceMethod describes one method for zero-reflection registration. Stream
// selects the streaming branch: the method registers as a grpc.StreamDesc and
// StreamHandler drives the stream, with the direction flags describing its
// shape for reflection. A pure-unary method leaves the stream fields zero.
type ServiceMethod struct {
	Name    string
	NewReq  func() any
	Handler UnaryHandler

	Stream        bool
	StreamHandler grpc.StreamHandler
	ServerStreams bool
	ClientStreams bool
}

// RegisterServiceWith registers a multi-method unary gRPC service with direct
// handlers. Middleware is applied per method via PrebuildHandler.
func RegisterServiceWith(s *Server, serviceName string, svr any, methods []struct {
	Name    string
	NewReq  func() any
	Handler UnaryHandler
}) {
	ms := make([]ServiceMethod, 0, len(methods))
	for _, m := range methods {
		ms = append(ms, ServiceMethod{Name: m.Name, NewReq: m.NewReq, Handler: m.Handler})
	}
	registerService(s, serviceName, svr, ms)
}

// RegisterServiceWithStreams registers a service whose methods may be unary or
// streaming, all in one ServiceDesc so a mixed service registers exactly once
// (grpc-go rejects a duplicate service name). Streaming is additive: pure-unary
// services keep using RegisterServiceWith unchanged, and only services that
// contain a stream method switch to this entry point.
func RegisterServiceWithStreams(s *Server, serviceName string, svr any, methods []ServiceMethod) {
	registerService(s, serviceName, svr, methods)
}

func registerService(s *Server, serviceName string, svr any, methods []ServiceMethod) {
	desc := &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*any)(nil),
	}
	for _, m := range methods {
		md := m
		fullMethod := "/" + serviceName + "/" + md.Name

		if md.Stream {
			desc.Streams = append(desc.Streams, grpc.StreamDesc{
				StreamName:    md.Name,
				Handler:       md.StreamHandler,
				ServerStreams: md.ServerStreams,
				ClientStreams: md.ClientStreams,
			})
			continue
		}

		op, _ := opFor(fullMethod)
		wrapped := s.PrebuildHandler(op, middleware.Handler(md.Handler))

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

