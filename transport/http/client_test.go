package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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

func TestHTTP(t *testing.T) {
	mu := sync.Mutex{}
	calls := 0
	var gotHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		gotHeader = r.Header.Get("name")
		mu.Unlock()
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	hc := NewClient(
		client.WithMiddleware(headerMW("cms")),
		client.WithRetry(client.RetryConfig{MaxAttempts: 2, InitialBackoff: 5 * time.Millisecond}),
	)
	resp, err := hc.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Fatalf("retry not observed: calls=%d want 2", calls)
	}
	if gotHeader != "cms" {
		t.Fatalf("injected header missing: got %q want cms", gotHeader)
	}
}
