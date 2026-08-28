package grpc

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
)

// headerMW is the protocol-agnostic middleware every test uses: it injects the
// "name" header into the outbound carrier, mirroring what cms does with its
// name + traceparent middleware.
func headerMW(name string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			client.OutgoingHeader(ctx).Set("name", name)
			return next(ctx, req)
		}
	}
}

func TestGrpcServiceConfig(t *testing.T) {
	cfg, err := grpcServiceConfig(client.RetryConfig{MaxAttempts: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"maxAttempts":3`,
		`"initialBackoff":"0.1s"`,
		`"maxBackoff":"1s"`,
		`"backoffMultiplier":2`,
		`"retryableStatusCodes":["UNAVAILABLE","DEADLINE_EXCEEDED"`,
		`"service":""`,
	} {
		if !strings.Contains(cfg, want) {
			t.Fatalf("service config %q missing %q", cfg, want)
		}
	}
	// The config must be accepted by grpc itself.
	if _, err := NewClient("127.0.0.1:1", client.WithRetry(client.RetryConfig{MaxAttempts: 3})); err != nil {
		t.Fatalf("NewClient with retry config: %v", err)
	}
}

// grpcSvc is the HandlerType marker interface grpc's RegisterService requires.
type grpcSvc interface{ call(context.Context) }

func (s *grpcServer) call(context.Context) {}

// grpcServer is a bare grpc server with one raw method, Call, that counts
// attempts, records incoming metadata, and can fail once with Unavailable.
type grpcServer struct {
	mu        sync.Mutex
	calls     int
	header    metadata.MD
	firstFail bool
}

func (s *grpcServer) start(t *testing.T) string {
	t.Helper()
	srv := grpc.NewServer()
	srv.RegisterService(&grpc.ServiceDesc{
		ServiceName: "test.Grpc",
		HandlerType: (*grpcSvc)(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Call",
			Handler: func(srv any, ctx context.Context, _ func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
				s := srv.(*grpcServer)
				s.mu.Lock()
				s.calls++
				s.header, _ = metadata.FromIncomingContext(ctx)
				first := s.calls == 1 && s.firstFail
				s.mu.Unlock()
				if first {
					return nil, status.Error(codes.Unavailable, "retry me")
				}
				return &emptypb.Empty{}, nil
			},
		}},
	}, s)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestGRPC(t *testing.T) {
	ts := &grpcServer{firstFail: true}
	addr := ts.start(t)

	conn, err := NewClient(addr,
		client.WithMiddleware(headerMW("cms")),
		client.WithRetry(client.RetryConfig{MaxAttempts: 2, InitialBackoff: 10 * time.Millisecond}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.Invoke(context.Background(), "/test.Grpc/Call", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatal(err)
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.calls != 2 {
		t.Fatalf("retry not observed: calls=%d want 2", ts.calls)
	}
	if got := ts.header.Get("name"); len(got) == 0 || got[0] != "cms" {
		t.Fatalf("injected header missing: got %v want cms", got)
	}
}
