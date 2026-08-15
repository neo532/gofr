package websocket

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
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
	return dialWSH(t, addr, path, nil)
}

func dialWSH(t *testing.T, addr, path string, hdr http.Header) *websocket.Conn {
	t.Helper()
	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), hdr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// dialWSRaw performs a WebSocket handshake over a raw TCP connection using the
// given request-line method, so tests can drive non-GET handshakes the way
// native HTTP would ("PUT /user/163 HTTP/1.1"). It returns a reader positioned
// at the first server frame.
func dialWSRaw(t *testing.T, addr, path, method string) *bufio.Reader {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	key := "dGhlIHNhbXBsZSBub25jZQ==" // "the sample nonce"
	req := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		method, path, addr, key)
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: method})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("%s handshake got %d, want %d", method, resp.StatusCode, http.StatusSwitchingProtocols)
	}
	return br
}

// readWSFrame reads one server-to-client WebSocket frame (unmasked) and returns
// its payload.
func readWSFrame(br *bufio.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return nil, err
	}
	n := uint64(hdr[1] & 0x7f)
	switch n {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		n = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		n = binary.BigEndian.Uint64(ext[:])
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(br, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func TestWebSocketEcho(t *testing.T) {
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/echo", func(ctx context.Context, conn *websocket.Conn) error {
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

func TestWebSocketClientIP(t *testing.T) {
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/ip", func(ctx context.Context, conn *websocket.Conn) error {
		tr, _ := transport.FromServerContext(ctx)
		return conn.WriteMessage(websocket.TextMessage, []byte(tr.ClientIP()))
	})

	addr, stop := startServer(t, srv)
	defer stop()
	_, port, _ := net.SplitHostPort(addr)

	u := url.URL{Scheme: "ws", Host: "127.0.0.1:" + port, Path: "/ip"}

	// Default simple mode: XFF honored.
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{"X-Forwarded-For": {"203.0.113.7"}})
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "203.0.113.7" {
		t.Fatalf("XFF: got %q, want %q", string(got), "203.0.113.7")
	}
	conn.Close()

	// No headers → direct peer address.
	conn2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, got2, err := conn2.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != "127.0.0.1" {
		t.Fatalf("RemoteAddr: got %q, want %q", string(got2), "127.0.0.1")
	}
	conn2.Close()
}

func TestWebSocketPathParam(t *testing.T) {
	// A :param route (as generated from a google.api.http annotation like
	// { get: "/room/{id}" }) must match the concrete request path.
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/room/:id", func(ctx context.Context, conn *websocket.Conn) error {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, msg)
	})

	addr, stop := startServer(t, srv)
	defer stop()

	conn := dialWS(t, addr, "/room/42")
	if err := conn.WriteMessage(websocket.TextMessage, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi" {
		t.Fatalf("got %q, want %q", string(got), "hi")
	}
}

func TestWebSocketNativeMethod(t *testing.T) {
	// Two methods sharing a path (get + put on "/user/{userId}", mirroring
	// google.api.http) coexist. The request-line method selects which one runs,
	// exactly as native HTTP: a GET handshake reaches the get: method, a PUT
	// handshake the put: method.
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/user/:id", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte("get"))
	})
	srv.Handle(http.MethodPut, "/user/:id", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte("put"))
	})

	addr, stop := startServer(t, srv)
	defer stop()

	// Standard GET handshake (browsers, gorilla dialer) → GET route.
	conn := dialWS(t, addr, "/user/163")
	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "get" {
		t.Fatalf("GET handshake routed to %q, want %q", string(got), "get")
	}
	conn.Close()

	// "PUT /user/163 HTTP/1.1" request line → PUT route.
	br := dialWSRaw(t, addr, "/user/163", http.MethodPut)
	payload, err := readWSFrame(br)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "put" {
		t.Fatalf("PUT handshake routed to %q, want %q", string(payload), "put")
	}
}

func TestWebSocketMethodInPath(t *testing.T) {
	// A browser only ever sends a GET handshake, so the method token rides in
	// the path: connecting to "/PUT/user/1" selects the put: method while the
	// wire handshake stays "GET /PUT/user/1 HTTP/1.1". The gorilla dialer here
	// is exactly that browser-style GET handshake.
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/user/:id", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte("get"))
	})
	srv.Handle(http.MethodPut, "/user/:id", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte("put"))
	})

	addr, stop := startServer(t, srv)
	defer stop()

	assert := func(path, want string) {
		t.Helper()
		conn := dialWS(t, addr, path)
		_, got, err := conn.ReadMessage()
		conn.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("GET handshake to %s routed to %q, want %q", path, string(got), want)
		}
	}

	assert("/user/163", "get")
	assert("/PUT./user/163", "put")
}

func TestWebSocketMethodInPathSlashNotMisjudged(t *testing.T) {
	// A path that merely begins with a verb segment is a real annotated path,
	// not a method selector: "/PUT/user/1" must reach its own GET route instead
	// of being rewritten to PUT on "/user/1".
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/PUT/user/:id", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte("literal"))
	})

	addr, stop := startServer(t, srv)
	defer stop()

	conn := dialWS(t, addr, "/PUT/user/163")
	_, got, err := conn.ReadMessage()
	conn.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "literal" {
		t.Fatalf("GET handshake to /PUT/user/163 routed to %q, want %q (verb-segment path misrouted)", string(got), "literal")
	}
}

func TestWebSocketNonGETRouteRejectsGET(t *testing.T) {
	// A path whose only route carries a non-GET verb (e.g.
	// { put: "/user/{userId}" }) is reachable only by a handshake that uses
	// that verb, exactly as in native HTTP. A plain GET handshake gets 405.
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodPut, "/user/:id", func(ctx context.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, []byte("put"))
	})

	addr, stop := startServer(t, srv)
	defer stop()

	u := url.URL{Scheme: "ws", Host: addr, Path: "/user/163"}
	_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err == nil {
		t.Fatal("expected GET handshake to a PUT-only route to be rejected")
	}
}

func TestWebSocketDuplicateRoutePanics(t *testing.T) {
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/echo", func(ctx context.Context, conn *websocket.Conn) error { return nil })
	srv.Handle(http.MethodGet, "/echo", func(ctx context.Context, conn *websocket.Conn) error { return nil })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate method+path route")
		}
	}()
	_ = srv.Start(context.Background())
}

func TestWebSocketNotFound(t *testing.T) {
	srv := NewServer(Address(":0"))
	srv.Handle(http.MethodGet, "/echo", func(ctx context.Context, conn *websocket.Conn) error {
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

	srv := NewServer(Address(":0"),
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				mu.Lock()
				logged = true
				mu.Unlock()
				return next(ctx, req)
			}
		}),
	)
	srv.Handle(http.MethodGet, "/test", func(ctx context.Context, conn *websocket.Conn) error {
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

	srv := NewServer(Address(":0"))
	srv.UseWith("/test", func(next transport.Handler) transport.Handler {
		return func(ctx context.Context, req any) (any, error) {
			mu.Lock()
			logged = true
			mu.Unlock()
			return next(ctx, req)
		}
	})
	srv.Handle(http.MethodGet, "/test", func(ctx context.Context, conn *websocket.Conn) error {
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
	srv := NewServer(Address(":0"),
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				return nil, transportError("rejected")
			}
		}),
	)
	srv.Handle(http.MethodGet, "/reject", func(ctx context.Context, conn *websocket.Conn) error {
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
