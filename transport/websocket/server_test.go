package websocket

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/neo532/gofr/transport"
)

func startServer(t *testing.T, srv *Server) (addr string, stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)
	stop = func() {
		cancel()
		stopCtx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer scancel()
		srv.Stop(stopCtx)
		<-done
	}
	return srv.Addr(), stop
}

func dialWS(t *testing.T, addr, path string) *websocket.Conn {
	t.Helper()
	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestWebSocketEcho(t *testing.T) {
	srv := NewServer(":0")
	srv.Handle("/echo", func(ctx context.Context, conn *websocket.Conn) error {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
		return nil
	})

	addr, stop := startServer(t, srv)
	defer stop()

	conn := dialWS(t, addr, "/echo")

	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", string(got), "hello")
	}
}

func TestWebSocketNotFound(t *testing.T) {
	srv := NewServer(":0")
	srv.Handle("/echo", func(ctx context.Context, conn *websocket.Conn) error {
		return nil
	})

	addr, stop := startServer(t, srv)
	defer stop()

	u := url.URL{Scheme: "ws", Host: addr, Path: "/nonexistent"}
	_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestWebSocketMiddleware(t *testing.T) {
	var mu sync.Mutex
	logged := false

	srv := NewServer(":0",
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				mu.Lock()
				logged = true
				mu.Unlock()
				return next(ctx, req)
			}
		}),
	)
	srv.Handle("/test", func(ctx context.Context, conn *websocket.Conn) error {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, msg)
	})

	addr, stop := startServer(t, srv)
	defer stop()

	conn := dialWS(t, addr, "/test")
	conn.WriteMessage(websocket.TextMessage, []byte("mw"))
	conn.Close()

	mu.Lock()
	ok := logged
	mu.Unlock()
	if !ok {
		t.Fatal("middleware was not called")
	}
}

func TestWebSocketUseWith(t *testing.T) {
	var mu sync.Mutex
	logged := false

	srv := NewServer(":0")
	srv.UseWith("/test", func(next transport.Handler) transport.Handler {
		return func(ctx context.Context, req any) (any, error) {
			mu.Lock()
			logged = true
			mu.Unlock()
			return next(ctx, req)
		}
	})
	srv.Handle("/test", func(ctx context.Context, conn *websocket.Conn) error {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, msg)
	})

	addr, stop := startServer(t, srv)
	defer stop()

	conn := dialWS(t, addr, "/test")
	conn.WriteMessage(websocket.TextMessage, []byte("usewith"))
	conn.Close()

	mu.Lock()
	ok := logged
	mu.Unlock()
	if !ok {
		t.Fatal("UseWith middleware was not called")
	}
}

func TestWebSocketMiddlewareRejects(t *testing.T) {
	srv := NewServer(":0",
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				return nil, transportError("rejected")
			}
		}),
	)
	srv.Handle("/reject", func(ctx context.Context, conn *websocket.Conn) error {
		return nil
	})

	addr, stop := startServer(t, srv)
	defer stop()

	u := url.URL{Scheme: "ws", Host: addr, Path: "/reject"}
	_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err == nil {
		t.Fatal("expected error when middleware rejects upgrade")
	}
}

// transportError is a simple error type for middleware rejection.
type transportError string

func (e transportError) Error() string { return string(e) }
