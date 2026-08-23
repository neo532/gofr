package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/neo532/gofr/transport"
)

func TestMiddlewareSeesRawRequest(t *testing.T) {
	var got string
	srv := NewServer(Address(":0"), Middleware(func(next transport.Handler) transport.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if htr, ok := tr.(*Transport); ok {
					if raw := htr.RawRequest(); raw != nil {
						got = raw.Method + " " + raw.URL.RequestURI()
					}
				}
			}
			return next(ctx, req)
		}
	}))
	HandleUnary(srv, "GET", "/raw",
		func(ctx context.Context, req *helloReq) (*helloReply, error) {
			return &helloReply{Message: "ok"}, nil
		},
		nil,
	)

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Get("http://" + addr + "/raw?page=2")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got != "GET /raw?page=2" {
		t.Fatalf("middleware saw %q, want %q", got, "GET /raw?page=2")
	}
}
