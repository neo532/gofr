package rpcx

import (
	"context"
	"net"
	"testing"
	"time"

	rpcxClient "github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/share"

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

// IPService reports the resolved client IP back through the reply.
type IPService struct{}

func (s *IPService) Get(ctx context.Context, args *HelloArgs, reply *HelloReply) error {
	tr, _ := transport.FromServerContext(ctx)
	if tr != nil {
		reply.Message = tr.ClientIP()
	}
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
	// Listen on ":0" yields "[::]:port"; force clients to 127.0.0.1 so the
	// server's observed peer address is deterministic (IPv4 loopback).
	_, port, err := net.SplitHostPort(lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return "127.0.0.1:" + port, func() { lis.Close() }
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

func TestRPCXClientIP(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterServiceWith(srv, "IPService", &IPService{})

	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXClient(t, addr)
	reply := &HelloReply{}

	// Default simple mode: metadata honored from any peer.
	ctx := context.WithValue(context.Background(), share.ReqMetaDataKey, map[string]string{"X-Forwarded-For": "203.0.113.7"})
	if err := c.Call(ctx, "IPService", "Get", &HelloArgs{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "203.0.113.7" {
		t.Fatalf("XFF: got %q, want %q", reply.Message, "203.0.113.7")
	}

	// No metadata → direct TCP peer.
	if err := c.Call(context.Background(), "IPService", "Get", &HelloArgs{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "127.0.0.1" {
		t.Fatalf("RemoteAddr: got %q, want %q", reply.Message, "127.0.0.1")
	}
}

func TestRPCXClientIPTrustedProxy(t *testing.T) {
	srv := NewServer(Address(":0"), TrustedProxies("10.0.0.0/8"))
	RegisterServiceWith(srv, "IPService", &IPService{})

	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXClient(t, addr)
	reply := &HelloReply{}

	// Peer 127.0.0.1 is not in the trusted 10.0.0.0/8 range → forged metadata ignored.
	ctx := context.WithValue(context.Background(), share.ReqMetaDataKey, map[string]string{"X-Forwarded-For": "203.0.113.7"})
	if err := c.Call(ctx, "IPService", "Get", &HelloArgs{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "127.0.0.1" {
		t.Fatalf("untrusted peer: got %q, want %q", reply.Message, "127.0.0.1")
	}
}

func TestRPCXRegisterService(t *testing.T) {
	srv := NewServer(Address(":0"))
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

	srv := NewServer(Address(":0"),
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

	srv := NewServer(Address(":0"))
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
