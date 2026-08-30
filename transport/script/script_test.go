package script

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

func TestScriptClientIP(t *testing.T) {
	ip := (&scriptTransport{}).ClientIP()
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsLoopback() {
		t.Fatalf("expected a real non-loopback IPv4, got %q", ip)
	}
	if ip4 := parsed.To4(); ip4 == nil {
		t.Fatalf("expected IPv4, got %q", ip)
	}
}

func TestScriptTransportCommand(t *testing.T) {
	tr := &scriptTransport{command: "greet", args: []string{"--name", "neo"}}
	if got, want := strings.Join(tr.Command(), " "), "greet --name neo"; got != want {
		t.Fatalf("Command() = %q, want %q", got, want)
	}
}

func TestScriptRunsMiddlewareWithCommand(t *testing.T) {
	called := false
	var command []string
	s := New(map[string]Func{
		"greet": func(ctx context.Context, args ...string) error {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return errors.New("no transport in context")
			}
			command = tr.(*scriptTransport).Command()
			return nil
		},
	}, Middleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			called = true
			return next(ctx, req)
		}
	}))

	if err := s.run(context.Background(), []string{"greet", "--name", "neo"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("middleware was not called")
	}
	if got, want := strings.Join(command, " "), "greet --name neo"; got != want {
		t.Fatalf("command seen by script = %q, want %q", got, want)
	}
}

func TestScriptMiddlewareError(t *testing.T) {
	boom := errors.New("boom")
	s := New(map[string]Func{
		"boom": func(ctx context.Context, args ...string) error {
			return boom
		},
	}, Middleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}))

	if err := s.run(context.Background(), []string{"boom"}); err == nil || !errors.Is(err, boom) {
		t.Fatalf("got error %v, want boom", err)
	}
}
