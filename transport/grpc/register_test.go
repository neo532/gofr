package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/middleware/manager"
	"github.com/neo532/gofr/transport"
)

// JSON codec for gRPC tests (non-protobuf messages).
type testCodec struct{}

func (testCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (testCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (testCodec) Name() string                       { return "json" }

func init() { encoding.RegisterCodec(testCodec{}) }

// test types.
type helloReq struct {
	Name string
}

type helloReply struct {
	Message string
}

// testSvc implements a simple gRPC service.
type testSvc struct{}

func (s testSvc) SayHello(ctx context.Context, req *helloReq) (*helloReply, error) {
	return &helloReply{Message: "Hello " + req.Name}, nil
}

func startGRPCServer(t *testing.T, srv *Server) (addr string, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()
	go func() { srv.Server.Serve(lis) }()
	time.Sleep(50 * time.Millisecond)
	return lis.Addr().String(), cancel
}

func dialGRPC(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func invoke(t *testing.T, conn *grpc.ClientConn, method string, req, reply any) {
	t.Helper()
	if err := conn.Invoke(context.Background(), method, req, reply); err != nil {
		t.Fatal(err)
	}
}

func TestGRPCRegisterServiceWith(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterServiceWith(srv, "test.Greeter", &testSvc{}, []struct {
		Name    string
		NewReq  func() any
		Handler UnaryHandler
	}{
		{
			Name:   "SayHello",
			NewReq: func() any { return &helloReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				return testSvc{}.SayHello(ctx, req.(*helloReq))
			},
		},
	})

	addr, stop := startGRPCServer(t, srv)
	defer stop()

	conn := dialGRPC(t, addr)
	reply := &helloReply{}
	invoke(t, conn, "/test.Greeter/SayHello", &helloReq{Name: "World"}, reply)
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestGRPCClientIP(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterServiceWith(srv, "test.Greeter", &testSvc{}, []struct {
		Name    string
		NewReq  func() any
		Handler UnaryHandler
	}{
		{
			Name:   "ClientIP",
			NewReq: func() any { return &helloReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				tr, _ := transport.FromServerContext(ctx)
				return &helloReply{Message: tr.ClientIP()}, nil
			},
		},
	})

	addr, stop := startGRPCServer(t, srv)
	defer stop()
	_, port, _ := net.SplitHostPort(addr)

	conn := dialGRPC(t, "127.0.0.1:"+port)
	reply := &helloReply{}

	// Default simple mode: x-forwarded-for metadata honored.
	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-forwarded-for", "203.0.113.7")
	if err := conn.Invoke(ctx, "/test.Greeter/ClientIP", &helloReq{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "203.0.113.7" {
		t.Fatalf("XFF: got %q, want %q", reply.Message, "203.0.113.7")
	}

	// No metadata → direct peer address.
	if err := conn.Invoke(context.Background(), "/test.Greeter/ClientIP", &helloReq{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "127.0.0.1" {
		t.Fatalf("RemoteAddr: got %q, want %q", reply.Message, "127.0.0.1")
	}
}

func TestGRPCMiddleware(t *testing.T) {
	var mu sync.Mutex
	logged := false

	srv := NewServer(Address(":0"),
		Middleware(func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				mu.Lock()
				logged = true
				mu.Unlock()
				return next(ctx, req)
			}
		}),
	)
	RegisterServiceWith(srv, "test.Greeter", &testSvc{}, []struct {
		Name    string
		NewReq  func() any
		Handler UnaryHandler
	}{
		{
			Name:   "SayHello",
			NewReq: func() any { return &helloReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				return testSvc{}.SayHello(ctx, req.(*helloReq))
			},
		},
	})

	addr, stop := startGRPCServer(t, srv)
	defer stop()

	conn := dialGRPC(t, addr)
	reply := &helloReply{}
	invoke(t, conn, "/test.Greeter/SayHello", &helloReq{Name: "MW"}, reply)

	mu.Lock()
	ok := logged
	mu.Unlock()
	if !ok {
		t.Fatal("middleware was not called")
	}
}

func TestGRPCUseWith(t *testing.T) {
	var mu sync.Mutex
	logged := false

	srv := NewServer(Address(":0"),
		WithMiddlewareManager(manager.NewMiddlewareManager(
			manager.NewExactMatcher("/test.Greeter/SayHello").Use(func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, req any) (any, error) {
					mu.Lock()
					logged = true
					mu.Unlock()
					return next(ctx, req)
				}
			}),
		)),
	)
	RegisterServiceWith(srv, "test.Greeter", &testSvc{}, []struct {
		Name    string
		NewReq  func() any
		Handler UnaryHandler
	}{
		{
			Name:   "SayHello",
			NewReq: func() any { return &helloReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				return testSvc{}.SayHello(ctx, req.(*helloReq))
			},
		},
	})

	addr, stop := startGRPCServer(t, srv)
	defer stop()

	conn := dialGRPC(t, addr)
	reply := &helloReply{}
	invoke(t, conn, "/test.Greeter/SayHello", &helloReq{Name: "UseWith"}, reply)

	mu.Lock()
	ok := logged
	mu.Unlock()
	if !ok {
		t.Fatal("UseWith middleware was not called")
	}
}

// startServerEx creates a server and returns the actual listening address.
// Used for testing Start/Stop lifecycle.
func startServerEx(t *testing.T, srv *Server) (addr string, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := srv.Server.Serve(lis); err != nil {
			t.Logf("Serve returned: %v", err)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	return lis.Addr().String(), func() { lis.Close() }
}

func TestGRPCLifecycle(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterServiceWith(srv, "test.Greeter", &testSvc{}, []struct {
		Name    string
		NewReq  func() any
		Handler UnaryHandler
	}{
		{
			Name:   "SayHello",
			NewReq: func() any { return &helloReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				return &helloReply{Message: fmt.Sprintf("Hello %s", req.(*helloReq).Name)}, nil
			},
		},
	})

	addr, stop := startServerEx(t, srv)
	defer stop()

	conn := dialGRPC(t, addr)
	reply := &helloReply{}
	invoke(t, conn, "/test.Greeter/SayHello", &helloReq{Name: "Lifecycle"}, reply)
	if reply.Message != "Hello Lifecycle" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello Lifecycle")
	}
}

func TestRegisterServiceWithStreams(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterServiceWithStreams(srv, "test.Mixed", &testSvc{}, []ServiceMethod{
		{
			Name:   "SayHello",
			NewReq: func() any { return &helloReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				return &helloReply{Message: "unary"}, nil
			},
		},
		{
			Name:          "Watch",
			Stream:        true,
			StreamHandler: func(srv any, stream grpc.ServerStream) error { return nil },
			ServerStreams: true,
		},
	})

	svc, ok := srv.GetServiceInfo()["test.Mixed"]
	if !ok {
		t.Fatal("service test.Mixed not registered")
	}
	if len(svc.Methods) != 2 {
		t.Fatalf("got %d methods, want 2 (SayHello unary + Watch stream)", len(svc.Methods))
	}
	seen := make(map[string]grpc.MethodInfo, len(svc.Methods))
	for _, m := range svc.Methods {
		seen[m.Name] = m
	}
	if u, ok := seen["SayHello"]; !ok || u.IsClientStream || u.IsServerStream {
		t.Fatalf("SayHello should be unary, got %+v", u)
	}
	if s, ok := seen["Watch"]; !ok || !s.IsServerStream || s.IsClientStream {
		t.Fatalf("Watch should be server-streaming, got %+v", s)
	}
}
