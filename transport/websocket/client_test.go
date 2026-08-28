package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

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

func TestWS(t *testing.T) {
	upgrader := websocket.Upgrader{}
	mu := sync.Mutex{}
	dialCount := 0
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		dialCount++
		gotHeader = r.Header.Get("name")
		mu.Unlock()
		if dialCount == 1 {
			http.Error(w, "not yet", http.StatusServiceUnavailable)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
	}))
	defer srv.Close()

	d := NewClient(
		client.WithMiddleware(headerMW("cms")),
		client.WithRetry(client.RetryConfig{MaxAttempts: 2, InitialBackoff: 5 * time.Millisecond}),
	)
	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")
	conn, _, err := d.DialContext(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	mu.Lock()
	defer mu.Unlock()
	if dialCount != 2 {
		t.Fatalf("dial retry not observed: count=%d want 2", dialCount)
	}
	if gotHeader != "cms" {
		t.Fatalf("injected header missing: got %q want cms", gotHeader)
	}
}
