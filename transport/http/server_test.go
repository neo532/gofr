package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/neo532/gofr/transport"
)

type helloReq struct {
	Name string `json:"name"`
	ID   int64  `json:"id"`
}

type helloReply struct {
	Message string `json:"message"`
}

type testService struct{}

func (s *testService) SayHello(ctx context.Context, req *helloReq) (*helloReply, error) {
	return &helloReply{Message: "Hello " + req.Name}, nil
}

func startServer(t *testing.T, srv *Server) (addr string, stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)
	_, port, _ := net.SplitHostPort(srv.lis.Addr().String())
	addr = "127.0.0.1:" + port
	stop = func() {
		cancel()
		<-done
	}
	return
}

func TestCustomRoute(t *testing.T) {
	srv := NewServer(Address(":0"))
	srv.GET("/hello/:Name", func(ctx Context) error {
		return ctx.Result(200, map[string]string{"greeting": "hello " + ctx.PathValue("Name")})
	})

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Get("http://" + addr + "/hello/GoFr")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply["greeting"] != "hello GoFr" {
		t.Fatalf("got %q, want %q", reply["greeting"], "hello GoFr")
	}
}

// TestRegisterUnaryPathParam mirrors the generated decoder shape:
// body binding plus int64 path-param injection, zero reflection.
func TestRegisterUnaryPathParam(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterUnary(srv, "PUT", "/api/v1/user/:userId",
		func(ctx context.Context, req *helloReq) (*helloReply, error) {
			return &helloReply{Message: fmt.Sprintf("id=%d name=%s", req.ID, req.Name)}, nil
		},
		func(ctx Context, req *helloReq) error {
			if err := ctx.Bind(req); err != nil {
				return err
			}
			req.ID, _ = strconv.ParseInt(ctx.PathValue("userId"), 10, 64)
			return nil
		},
	)

	addr, stop := startServer(t, srv)
	defer stop()

	req, err := http.NewRequest("PUT", "http://"+addr+"/api/v1/user/42",
		strings.NewReader(`{"name":"World"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "id=42 name=World" {
		t.Fatalf("got %q, want %q", reply.Message, "id=42 name=World")
	}
}

func TestServerMiddleware(t *testing.T) {
	var logged bool

	srv := NewServer(Address(":0"),
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req any) (any, error) {
				logged = true
				return next(ctx, req)
			}
		}),
	)
	RegisterUnary(srv, "POST", "/test.Greeter/SayHello",
		(&testService{}).SayHello,
		func(ctx Context, req *helloReq) error { return ctx.Bind(req) },
	)

	addr, stop := startServer(t, srv)
	defer stop()

	http.Post("http://"+addr+"/test.Greeter/SayHello",
		"application/json", strings.NewReader(`{"name":"x"}`))

	if !logged {
		t.Fatal("middleware was not called")
	}
}

func TestRegisterHandler(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterHandler(srv, "test.Greeter/SayHello", (&testService{}).SayHello)

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Post(
		"http://"+addr+"/test.Greeter/SayHello",
		"application/json",
		strings.NewReader(`{"name":"World"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestClientIP(t *testing.T) {
	newServer := func(opts ...ServerOption) (addr string, stop func()) {
		t.Helper()
		srv := NewServer(append([]ServerOption{Address(":0")}, opts...)...)
		srv.GET("/ip", func(ctx Context) error {
			return ctx.Result(200, map[string]string{"ip": ctx.ClientIP()})
		})
		return startServer(t, srv)
	}
	get := func(addr string, headers map[string]string) string {
		t.Helper()
		req, err := http.NewRequest("GET", "http://"+addr+"/ip", nil)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var reply map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
			t.Fatal(err)
		}
		return reply["ip"]
	}

	// Default (no TrustedProxies): headers from any peer are trusted.
	simpleAddr, stopSimple := newServer()
	defer stopSimple()
	if ip := get(simpleAddr, map[string]string{"X-Forwarded-For": "203.0.113.7, 10.0.0.1"}); ip != "203.0.113.7" {
		t.Fatalf("default XFF: got %q, want %q", ip, "203.0.113.7")
	}
	if ip := get(simpleAddr, map[string]string{"X-Real-IP": "203.0.113.8"}); ip != "203.0.113.8" {
		t.Fatalf("default X-Real-IP: got %q, want %q", ip, "203.0.113.8")
	}
	if ip := get(simpleAddr, map[string]string{"X-Forwarded-For": "not-an-ip"}); ip != "127.0.0.1" {
		t.Fatalf("invalid XFF fallback: got %q, want %q", ip, "127.0.0.1")
	}

	// Peer 127.0.0.1 IS a trusted proxy: headers are honored.
	trustedAddr, stopTrusted := newServer(TrustedProxies("127.0.0.1"))
	defer stopTrusted()
	if ip := get(trustedAddr, map[string]string{"X-Forwarded-For": "203.0.113.9"}); ip != "203.0.113.9" {
		t.Fatalf("trusted proxy XFF: got %q, want %q", ip, "203.0.113.9")
	}

	// Peer 127.0.0.1 is NOT a trusted proxy: forged headers are ignored.
	untrustedAddr, stopUntrusted := newServer(TrustedProxies("10.0.0.0/8"))
	defer stopUntrusted()
	if ip := get(untrustedAddr, map[string]string{"X-Forwarded-For": "203.0.113.9"}); ip != "127.0.0.1" {
		t.Fatalf("forged XFF: got %q, want %q", ip, "127.0.0.1")
	}
	if ip := get(untrustedAddr, map[string]string{"X-Real-IP": "203.0.113.9"}); ip != "127.0.0.1" {
		t.Fatalf("forged X-Real-IP: got %q, want %q", ip, "127.0.0.1")
	}
	if ip := get(untrustedAddr, nil); ip != "127.0.0.1" {
		t.Fatalf("RemoteAddr fallback: got %q, want %q", ip, "127.0.0.1")
	}
}

func TestRegisterUnary(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterUnary(srv, "GET", "/api/v1/hello/:name",
		(&testService{}).SayHello,
		func(ctx Context, req *helloReq) error {
			req.Name = ctx.PathValue("name")
			return nil
		},
	)

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Get("http://" + addr + "/api/v1/hello/World")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}
