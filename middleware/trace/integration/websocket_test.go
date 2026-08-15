package integration

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/gorilla/websocket"

	gofrTrace "github.com/neo532/gofr/middleware/trace"
	gofrws "github.com/neo532/gofr/transport/websocket"
)

func TestWebSocketTracePropagation(t *testing.T) {
	srv := gofrws.NewServer(gofrws.Address(":0"), gofrws.Middleware(gofrTrace.Server()))
	srv.Handle(http.MethodGet, "/trace", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte(gofrTrace.TraceID(ctx)))
	})

	addr, stop := start(t, srv)
	defer stop()

	u := url.URL{Scheme: "ws", Host: addr, Path: "/trace"}

	// Incoming traceparent header → handler sees the same trace ID.
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{"traceparent": {traceparentHeader}})
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != wantTraceID {
		t.Fatalf("incoming traceparent: got %q, want %q", string(got), wantTraceID)
	}
	conn.Close()

	// No header → generated valid 32-hex ID.
	conn2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, got2, err := conn2.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if len(got2) != 32 || string(got2) == "00000000000000000000000000000000" {
		t.Fatalf("generated trace ID: got %q, want 32-hex non-zero", string(got2))
	}
	conn2.Close()
}
