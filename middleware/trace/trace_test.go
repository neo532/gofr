package trace

import (
	"context"
	"encoding/hex"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

const testTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

// mapHeader adapts map[string]string to transport.Header.
type mapHeader map[string]string

func (m mapHeader) Get(key string) string { return map[string]string(m)[key] }
func (m mapHeader) Set(key, value string) { map[string]string(m)[key] = value }
func (m mapHeader) Keys() []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// fakeTransport is a minimal transport.Transporter for middleware tests.
type fakeTransport struct {
	reqHeader   transport.Header
	replyHeader transport.Header
}

func (t *fakeTransport) Kind() transport.Kind           { return transport.KindHTTP }
func (t *fakeTransport) Endpoint() string                { return "test" }
func (t *fakeTransport) Operation() string               { return "/test" }
func (t *fakeTransport) RequestHeader() transport.Header  { return t.reqHeader }
func (t *fakeTransport) ReplyHeader() transport.Header    { return t.replyHeader }
func (t *fakeTransport) App() transport.App              { return nil }
func (t *fakeTransport) ClientIP() string                { return "127.0.0.1" }

func runServerMiddleware(t *testing.T, hdr transport.Header) (traceID string, reply traceparent, err error) {
	t.Helper()
	chain := middleware.Chain(Server())
	replyHdr := mapHeader{}
	ctx := transport.NewServerContext(context.Background(), &fakeTransport{
		reqHeader:   hdr,
		replyHeader: replyHdr,
	})
	h := chain(func(ctx context.Context, req any) (any, error) {
		return TraceID(ctx), nil
	})
	out, err := h(ctx, nil)
	return out.(string), traceparent(replyHdr.Get("traceparent")), err
}

// traceparent is a parsed W3C traceparent header.
type traceparent string

func (tp traceparent) traceID() string {
	if len(tp) < 3+32 {
		return ""
	}
	return string(tp[3 : 3+32])
}

func (tp traceparent) spanID() string {
	if len(tp) < 3+32+1+16 {
		return ""
	}
	return string(tp[3+32+1 : 3+32+1+16])
}

func TestServerMiddlewareExtractsIncomingTrace(t *testing.T) {
	got, reply, err := runServerMiddleware(t, mapHeader{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != testTraceID {
		t.Fatalf("got %q, want %q", got, testTraceID)
	}
	if reply.traceID() != testTraceID {
		t.Fatalf("reply traceparent trace ID %q, want %q", reply, testTraceID)
	}
}

// TestServerMiddlewareMintsOwnSpan verifies that with an incoming traceparent
// but no real SDK (no-op tracer), the middleware keeps the parent's trace id for
// cross-service continuity yet mints its own span id — each service hop must be
// its own span, not inherit the upstream span.
func TestServerMiddlewareMintsOwnSpan(t *testing.T) {
	const parentSpanID = "00f067aa0ba902b7"
	_, reply, err := runServerMiddleware(t, mapHeader{
		"traceparent": "00-" + testTraceID + "-" + parentSpanID + "-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.traceID() != testTraceID {
		t.Fatalf("trace id %q, want %q (must keep parent trace id)", reply.traceID(), testTraceID)
	}
	if reply.spanID() == parentSpanID {
		t.Fatalf("span id must differ from the parent, got %q", reply.spanID())
	}
	if len(reply.spanID()) != 16 || reply.spanID() == "0000000000000000" {
		t.Fatalf("span id must be a fresh 16-hex id, got %q", reply.spanID())
	}
}

func TestServerMiddlewareGeneratesRoot(t *testing.T) {
	got, reply, err := runServerMiddleware(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("expected 32-hex trace ID, got %q", got)
	}
	if got == "00000000000000000000000000000000" {
		t.Fatal("trace ID must not be all zeros")
	}
	if reply.traceID() != got {
		t.Fatalf("reply traceparent trace ID %q, want %q", reply, got)
	}
}

func TestTraceparent(t *testing.T) {
	const tp = "00-" + testTraceID + "-00f067aa0ba902b7-01"
	chain := middleware.Chain(Server())
	ctx := transport.NewServerContext(context.Background(), &fakeTransport{
		reqHeader: mapHeader{"traceparent": tp},
	})
	h := chain(func(ctx context.Context, req any) (any, error) {
		return Traceparent(ctx), nil
	})
	out, err := h(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.(string); got != tp {
		t.Fatalf("traceparent = %q, want %q", got, tp)
	}
}

func TestTraceparentRoot(t *testing.T) {
	chain := middleware.Chain(Server())
	ctx := transport.NewServerContext(context.Background(), &fakeTransport{reqHeader: nil})
	h := chain(func(ctx context.Context, req any) (any, error) {
		return Traceparent(ctx), nil
	})
	out, err := h(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.(string); got != "" {
		t.Fatalf("root traceparent = %q, want empty", got)
	}
}

func TestServerMiddlewarePropagatesError(t *testing.T) {
	chain := middleware.Chain(Server())
	ctx := transport.NewServerContext(context.Background(), &fakeTransport{reqHeader: nil})
	h := chain(func(ctx context.Context, req any) (any, error) {
		return nil, context.DeadlineExceeded
	})
	if _, err := h(ctx, nil); err != context.DeadlineExceeded {
		t.Fatalf("got %v, want DeadlineExceeded", err)
	}
}

func mustTraceID(t *testing.T, s string) trace.TraceID {
	t.Helper()
	var id trace.TraceID
	if _, err := hex.Decode(id[:], []byte(s)); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestInjectExtractRoundTrip(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    mustTraceID(t, testTraceID),
		SpanID:     trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

	carrier := mapHeader{}
	Inject(ctx, carrier)
	if carrier.Get("traceparent") == "" {
		t.Fatal("Inject did not write traceparent")
	}

	got := TraceID(Extract(context.Background(), carrier))
	if got != testTraceID {
		t.Fatalf("round-trip got %q, want %q", got, testTraceID)
	}
}

func TestTraceIDEmptyWithoutTrace(t *testing.T) {
	if got := TraceID(context.Background()); got != "" {
		t.Fatalf("expected empty trace ID, got %q", got)
	}
}
