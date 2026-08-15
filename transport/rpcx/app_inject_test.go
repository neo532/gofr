package rpcx

import (
	"context"
	"testing"

	"github.com/neo532/gofr/transport"
	"github.com/neo532/gokit/logger"
)

type fakeApp struct{}

func (fakeApp) Logger() logger.ILogger { return nil }

// TestRPCXAppInjection verifies that an App injected via Server.App (as
// gofr.App.Run does) is visible to middleware through the Transporter. This
// guards the rpcx middlewarePlugin against capturing s.app before App() runs.
func TestRPCXAppInjection(t *testing.T) {
	var gotApp transport.App
	srv := NewServer(Address(":0"),
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				if tr, ok := transport.FromServerContext(ctx); ok {
					gotApp = tr.App()
				}
				return next(ctx, req)
			}
		}),
	)
	srv.App(fakeApp{})
	RegisterServiceWith(srv, "HelloService", &HelloService{})

	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXClient(t, addr)
	reply := &HelloReply{}
	if err := c.Call(context.Background(), "HelloService", "SayHello", &HelloArgs{Name: "x"}, reply); err != nil {
		t.Fatal(err)
	}
	if gotApp == nil {
		t.Fatal("App not injected into rpcx Transporter: tr.App() == nil")
	}
}
