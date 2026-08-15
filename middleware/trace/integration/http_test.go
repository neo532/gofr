package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	gofrTrace "github.com/neo532/gofr/middleware/trace"
	ghttp "github.com/neo532/gofr/transport/http"
)

type httpReq struct {
	Name string `json:"name"`
}

type httpReply struct {
	Message string `json:"message"`
}

func TestHTTPTracePropagation(t *testing.T) {
	srv := ghttp.NewServer(ghttp.Address(":0"), ghttp.Middleware(gofrTrace.Server()))
	ghttp.RegisterUnary(srv, "POST", "/trace",
		func(ctx context.Context, req *httpReq) (*httpReply, error) {
			return &httpReply{Message: gofrTrace.TraceID(ctx)}, nil
		},
		func(ctx ghttp.Context, req *httpReq) error { return ctx.Bind(req) },
	)

	addr, stop := start(t, srv)
	defer stop()

	post := func(header map[string]string) string {
		t.Helper()
		req, err := http.NewRequest("POST", "http://"+addr+"/trace", strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range header {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var reply httpReply
		if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
			t.Fatal(err)
		}
		return reply.Message
	}

	if got := post(map[string]string{"traceparent": traceparentHeader}); got != wantTraceID {
		t.Fatalf("incoming traceparent: got %q, want %q", got, wantTraceID)
	}

	if got := post(nil); len(got) != 32 || got == "00000000000000000000000000000000" {
		t.Fatalf("generated trace ID: got %q, want 32-hex non-zero", got)
	}
}
