package rpcx

import (
	"context"
	"testing"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// replySubscriber mirrors the structural interface kit/middleware uses to defer
// request-replay logging to the transport's PostCall.
type replySubscriber interface {
	OnReply(func(reply any, err error))
}

// TestRPCXReplyDeliveredToMiddleware verifies the rpcx plugin delivers the real
// reply to an OnReply subscriber registered by a middleware during PreCall.
// rpcx writes the reply into the pointer only after PreCall, so the middleware
// chain sees just the decoded request; this guards replay's deferred logging.
func TestRPCXReplyDeliveredToMiddleware(t *testing.T) {
	var gotReply any
	var gotErr error

	srv := NewServer(Address(":0"),
		Middleware(func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				if tr, ok := transport.FromServerContext(ctx); ok {
					if sub, ok := tr.(replySubscriber); ok {
						sub.OnReply(func(reply any, err error) {
							gotReply = reply
							gotErr = err
						})
					}
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
	if err := c.Call(context.Background(), "HelloService", "SayHello", &HelloArgs{Name: "World"}, reply); err != nil {
		t.Fatal(err)
	}
	if gotErr != nil {
		t.Fatalf("OnReply got err %v, want nil", gotErr)
	}
	hr, ok := gotReply.(*HelloReply)
	if !ok || hr.Message != "Hello World" {
		t.Fatalf("OnReply reply = %#v, want *HelloReply{Message: Hello World}", gotReply)
	}
}
