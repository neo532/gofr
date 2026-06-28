package rpcx

import (
	"context"
	"net"
	"testing"
	"time"

	rpcxClient "github.com/smallnest/rpcx/client"

	"github.com/neo532/gofr/transport"
)

// Args and Reply must be exported (capital first letter) for rpcx reflection.
type HelloArgs struct {
	Name string
}

type HelloReply struct {
	Message string
}

// HelloService is rpcx-compatible: method(ctx, *Args, *Reply) error.
type HelloService struct{}

func (s *HelloService) SayHello(ctx context.Context, args *HelloArgs, reply *HelloReply) error {
	reply.Message = "Hello " + args.Name
	return nil
}

func newTestServer(t *testing.T, srv *Server) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.ServeListener("tcp", lis)
	time.Sleep(100 * time.Millisecond)
	return lis.Addr().String(), func() { lis.Close() }
}

func newRPCXClient(t *testing.T, addr string) *rpcxClient.OneClient {
	t.Helper()
	d, err := rpcxClient.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		t.Fatal(err)
	}
	c := rpcxClient.NewOneClient(rpcxClient.Failtry, rpcxClient.RandomSelect, d, rpcxClient.DefaultOption)
	t.Cleanup(func() { c.Close() })
	return c
}

func TestRPCXRegisterService(t *testing.T) {
	srv := NewServer(":0")
	RegisterServiceWith(srv, "HelloService", &HelloService{})

	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXClient(t, addr)
	reply := &HelloReply{}
	if err := c.Call(context.Background(), "HelloService", "SayHello", &HelloArgs{Name: "World"}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestRPCXMiddleware(t *testing.T) {
	var logged bool

	srv := NewServer(":0",
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				logged = true
				return next(ctx, req)
			}
		}),
	)
	RegisterServiceWith(srv, "HelloService", &HelloService{})

	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXClient(t, addr)
	reply := &HelloReply{}
	if err := c.Call(context.Background(), "HelloService", "SayHello", &HelloArgs{Name: "MW"}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello MW" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello MW")
	}
	if !logged {
		t.Fatal("middleware was not called")
	}
}

func TestRPCXUseWith(t *testing.T) {
	var logged bool

	srv := NewServer(":0")
	srv.UseWith("/HelloService/SayHello", func(next transport.Handler) transport.Handler {
		return func(ctx context.Context, req any) (any, error) {
			logged = true
			return next(ctx, req)
		}
	})
	RegisterServiceWith(srv, "HelloService", &HelloService{})

	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXClient(t, addr)
	reply := &HelloReply{}
	if err := c.Call(context.Background(), "HelloService", "SayHello", &HelloArgs{Name: "x"}, reply); err != nil {
		t.Fatal(err)
	}
	if !logged {
		t.Fatal("UseWith middleware was not called")
	}
}
