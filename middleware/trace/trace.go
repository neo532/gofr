package trace

import (
	"context"
	"encoding/binary"
	"math/rand/v2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// propagator is gofr's default text map propagator: W3C Trace Context plus
// baggage. It is set explicitly rather than read from otel.GetTextMapPropagator()
// because OTel Go's global propagator defaults to a no-op until an app calls
// SetTextMapPropagator.
var propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// Option configures the Server middleware.
type Option func(*options)

type options struct {
	tp         trace.TracerProvider
	propagator propagation.TextMapPropagator
	tracerName string
}

// WithTracerProvider overrides the tracer provider. When unset, gofr resolves
// otel.GetTracerProvider() per request, so an SDK configured after the
// middleware is registered is still picked up. Without an SDK, spans are no-ops.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(o *options) { o.tp = tp }
}

// WithPropagator overrides the text map propagator. Defaults to W3C trace
// context + baggage.
func WithPropagator(p propagation.TextMapPropagator) Option {
	return func(o *options) { o.propagator = p }
}

// WithTracerName sets the tracer name. Defaults to "gofr".
func WithTracerName(name string) Option {
	return func(o *options) { o.tracerName = name }
}

// Server returns a middleware that, for each request, extracts the incoming
// trace context from the request headers, starts a server span named after the
// operation, and passes the span's context to the handler. The handler can read
// the active trace ID via TraceID(ctx), and downstream calls can propagate the
// context via Inject. The trace context is also injected into the reply header,
// so the response carries the same trace ID back to the caller.
func Server(opts ...Option) middleware.Middleware {
	o := &options{tracerName: "gofr"}
	for _, opt := range opts {
		opt(o)
	}
	if o.propagator == nil {
		o.propagator = propagator
	}

	return func(next transport.Handler) transport.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return next(ctx, req)
			}
			if hdr := tr.RequestHeader(); hdr != nil {
				if tp := hdr.Get("traceparent"); tp != "" {
					ctx = context.WithValue(ctx, traceparentKey{}, tp)
				}
				ctx = o.propagator.Extract(ctx, hdr)
			}
			parent := trace.SpanContextFromContext(ctx)
			tp := o.tp
			if tp == nil {
				tp = otel.GetTracerProvider()
			}
			ctx, span := tp.Tracer(o.tracerName).Start(ctx, tr.Operation(),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(attribute.String("client.ip", tr.ClientIP())),
			)
			defer span.End()
			// Without a real SDK the tracer is a no-op. OTel's no-op Start carries
			// the incoming remote (non-recording) span forward instead of minting a
			// new one, so the span context here may be the parent's — the same span
			// id the upstream service reported. Detect that (invalid context, or the
			// parent unchanged) and mint a real per-hop span context: keep the parent
			// trace id so trace continuity across services holds, but a fresh span id
			// so each service hop is its own span.
			sc := span.SpanContext()
			if !sc.IsValid() || (parent.IsValid() && sc.SpanID() == parent.SpanID()) {
				cfg := trace.SpanContextConfig{
					TraceID:    parent.TraceID(),
					SpanID:     newSpanID(),
					TraceFlags: trace.FlagsSampled,
				}
				if !parent.IsValid() {
					cfg.TraceID = newTraceID()
				}
				ctx = trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(cfg))
			}

			// Propagate the trace context back on the response so callers can
			// chain the trace and verify the same trace ID round-trips.
			if hdr := tr.ReplyHeader(); hdr != nil {
				o.propagator.Inject(ctx, hdr)
			}

			out, err := next(ctx, req)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return out, err
		}
	}
}

// Extract returns a context carrying the trace context read from carrier.
// Used at the start of a downstream call to pick up an inbound trace context.
func Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return propagator.Extract(ctx, carrier)
}

// Inject writes the current trace context from ctx into carrier. The carrier
// can be any propagation.TextMapCarrier (transport.Header, http.Header via
// propagation.HeaderCarrier, map[string]string via propagation.MapCarrier).
func Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	propagator.Inject(ctx, carrier)
}

func newTraceID() trace.TraceID {
	var id trace.TraceID
	binary.LittleEndian.PutUint64(id[0:8], rand.Uint64())
	binary.LittleEndian.PutUint64(id[8:16], rand.Uint64())
	return id
}

func newSpanID() trace.SpanID {
	var id trace.SpanID
	binary.LittleEndian.PutUint64(id[:], rand.Uint64())
	return id
}

// CarrySpan copies the span carried by src onto dst and returns the result.
// Used by transports whose handlers receive a pre-existing context that
// middleware-installed context changes cannot reach (e.g. rpcx), so the trace
// span still flows to the handler.
func CarrySpan(dst, src context.Context) context.Context {
	if span := trace.SpanFromContext(src); span.SpanContext().IsValid() {
		return trace.ContextWithSpan(dst, span)
	}
	return dst
}

// WithBootTrace returns a copy of ctx carrying a fresh sampled span context
// with randomly generated trace and span IDs. Services that run without an
// OTel SDK use it to synthesize a boot trace context at startup, so logs
// emitted outside any request still report a trace ID.
func WithBootTrace(ctx context.Context) context.Context {
	return trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    newTraceID(),
		SpanID:     newSpanID(),
		TraceFlags: trace.FlagsSampled,
	}))
}

// TraceID returns the active trace ID as a 32-hex string, or "" when no trace
// is active.
func TraceID(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanID returns the active span ID as a 16-hex string, or "" when no trace is
// active.
func SpanID(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.SpanID().String()
	}
	return ""
}

// traceparentKey carries the raw inbound traceparent header from the Server
// middleware into the handler context, so loggers can attach a "traceparent"
// field carrying the caller's trace context (trace id + parent span id + flags)
// and log entries can be chained across service hops.
type traceparentKey struct{}

// Traceparent returns the raw inbound W3C traceparent header value that started
// the current request span, or "" for a root span. The value is captured from
// the request header before extraction.
func Traceparent(ctx context.Context) string {
	if tp, ok := ctx.Value(traceparentKey{}).(string); ok {
		return tp
	}
	return ""
}
