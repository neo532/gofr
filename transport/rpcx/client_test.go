package rpcx

import (
	"context"
	"net"
	"sync"
	"testing"

	rpcxServer "github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
)

// headerMW injects the "name" header into the outbound carrier, mirroring what
// cms does with its name + traceparent middleware.
func headerMW(name string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			client.OutgoingHeader(ctx).Set("name", name)
			return next(ctx, req)
		}
	}
}

// rpcxSvc is a bare rpcx service whose Call reads the request metadata.
type rpcxSvc struct {
	mu     sync.Mutex
	calls  int
	header map[string]string
}

func (s *rpcxSvc) Call(ctx context.Context, _ *emptypb.Empty, _ *emptypb.Empty) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.header, _ = ctx.Value(share.ReqMetaDataKey).(map[string]string)
	return nil
}

// TestRPCX verifies the middleware-injected header reaches the server as rpcx
// request metadata. Retry is rpcx's native Failtry mode: it retries only
// network-level failures, never a service-returned error, so there is nothing
// to assert here beyond the header.
func TestRPCX(t *testing.T) {
	svc := &rpcxSvc{}
	srv := rpcxServer.NewServer()
	if err := srv.RegisterName("test.Rpcx", svc, ""); err != nil {
		t.Fatal(err)
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.ServeListener("tcp", lis)
	t.Cleanup(func() { srv.Close() })

	xc, err := NewClient("test.Rpcx", []string{lis.Addr().String()},
		client.WithMiddleware(headerMW("cms")), client.WithRetry(client.RetryConfig{MaxAttempts: 2}))
	if err != nil {
		t.Fatal(err)
	}
	defer xc.Close()

	if err := xc.Call(context.Background(), "Call", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatal(err)
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.calls != 1 {
		t.Fatalf("unexpected calls: %d", svc.calls)
	}
	if got := svc.header["name"]; got != "cms" {
		t.Fatalf("injected header missing: got %q want cms", got)
	}
}
