package grpc

import (
	"context"
	"fmt"
	"net"
	"reflect"

	"google.golang.org/grpc"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// ServerOption configures the gRPC server.
type ServerOption func(*Server)

// Address sets the listen address.
func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

// Middleware registers global middlewares applied to all methods.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.mwManager.Use(m...) }
}

// Server wraps grpc.Server and implements transport.Server with middleware.
type Server struct {
	*grpc.Server
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

// NewServer creates a gRPC server.
func NewServer(address string, opts ...grpc.ServerOption) *Server {
	s := &Server{
		Server:    grpc.NewServer(opts...),
		address:   address,
		mwManager: newMiddlewareManager(),
	}
	return s
}

// NewServerWith creates a gRPC server with gofr options.
func NewServerWith(address string, opts ...ServerOption) *Server {
	s := &Server{
		Server:    grpc.NewServer(),
		address:   address,
		mwManager: newMiddlewareManager(),
	}
	for _, o := range opts {
		o(s)
	}
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

// PrebuildHandler pre-computes middleware chain for a gRPC method.
func (s *Server) PrebuildHandler(fullMethod string, fn transport.Handler) transport.Handler {
	matched := s.mwManager.Match(fullMethod)
	return middleware.Chain(matched...)(fn)
}

// Start implements transport.Server.
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.lis = lis

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	return s.Server.Serve(lis)
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
type UnaryHandler func(ctx context.Context, req interface{}) (interface{}, error)

// RegisterServiceWith registers a multi-method gRPC service with direct handlers.
// Middleware is applied per method via PrebuildHandler.
func RegisterServiceWith(s *Server, serviceName string, svr interface{}, methods []struct {
	Name    string
	NewReq  func() interface{}
	Handler UnaryHandler
}) {
	desc := &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*interface{})(nil),
	}
	for _, m := range methods {
		md := m
		fullMethod := "/" + serviceName + "/" + md.Name
		wrapped := s.PrebuildHandler(fullMethod, transport.Handler(md.Handler))

		desc.Methods = append(desc.Methods, grpc.MethodDesc{
			MethodName: md.Name,
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := md.NewReq()
				if err := dec(req); err != nil {
					return nil, err
				}

				if interceptor != nil {
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: fullMethod,
					}
					return interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
						return wrapped(ctx, req)
					})
				}
				return wrapped(ctx, req)
			},
		})
	}
	s.Server.RegisterService(desc, svr)
}

// RegisterService registers a service from transport.ServiceDesc onto the gRPC server.
// Deprecated: use RegisterServiceWith and generated _grpc.pb.go for zero-reflection.
func RegisterService(srv *Server, desc *transport.ServiceDesc, svr interface{}) {
	grpcDesc := &grpc.ServiceDesc{
		ServiceName: desc.Name,
		HandlerType: (*interface{})(nil),
	}
	for _, m := range desc.Methods {
		md := m
		fullMethod := fmt.Sprintf("/%s/%s", desc.Name, md.Name)
		grpcDesc.Methods = append(grpcDesc.Methods, grpc.MethodDesc{
			MethodName: md.Name,
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := md.NewRequest()
				if err := dec(req); err != nil {
					return nil, err
				}
				if interceptor != nil {
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: fullMethod,
					}
					return interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
						return callMethod(srv, ctx, req, md.Name)
					})
				}
				return callMethod(srv, ctx, req, md.Name)
			},
		})
	}
	srv.Server.RegisterService(grpcDesc, svr)
}

// callMethod uses reflection to dispatch. Retained for backward compatibility only.
func callMethod(srv interface{}, ctx context.Context, req interface{}, name string) (interface{}, error) {
	srvVal := reflect.ValueOf(srv)
	method := srvVal.MethodByName(name)
	if !method.IsValid() {
		return nil, fmt.Errorf("gofr: service has no method %q", name)
	}
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(req),
	})
	var err error
	if len(results) > 1 && !results[1].IsNil() {
		err = results[1].Interface().(error)
	}
	return results[0].Interface(), err
}
