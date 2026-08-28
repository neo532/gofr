// Package client defines the shared outbound-client contract: the middleware
// and retry knobs every protocol client accepts, the outbound header carrier,
// and the generic FollowClient that keeps a typed client wired to the latest
// registry instance set.
package client

import (
	"context"
	"time"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// outgoingClientHeaderKey carries the outbound request header carrier a client
// exposes to its middleware for the current call.
type outgoingClientHeaderKey struct{}

// WithOutgoingHeader stores hdr as the outbound request header carrier of the
// current client call. Each protocol client installs it before running the
// middleware chain.
func WithOutgoingHeader(ctx context.Context, hdr transport.Header) context.Context {
	return context.WithValue(ctx, outgoingClientHeaderKey{}, hdr)
}

// OutgoingHeader returns the outbound request header carrier a client installed
// for the current call, or nil outside a client call. Middleware inject
// outbound headers (traceparent, name, auth, ...) by mutating it.
func OutgoingHeader(ctx context.Context) transport.Header {
	hdr, _ := ctx.Value(outgoingClientHeaderKey{}).(transport.Header)
	return hdr
}

// RetryConfig is the generic retry policy shared by all protocol clients. Each
// protocol translates it to its native retry mechanism: grpc's retryPolicy, an
// HTTP retry RoundTripper, rpcx's Failtry mode, a websocket dial retry.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts including the original
	// call. Must be >= 2; defaults to 3.
	MaxAttempts int
	// InitialBackoff is the base backoff before the first retry. Default 100ms.
	InitialBackoff time.Duration
	// MaxBackoff caps the exponential backoff growth. Default 1s.
	MaxBackoff time.Duration
	// BackoffMultiplier grows the backoff between retries. Default 2.0.
	BackoffMultiplier float64
	// Retryable lists the retryable conditions in protocol-neutral form. For
	// grpc it is the uppercase proto status-code names ("UNAVAILABLE"); for
	// http the explicit status codes ("502"). A protocol that cannot express
	// per-condition retry (rpcx, websocket) ignores it.
	Retryable []string
}

// Normalized returns r with unset numeric fields filled from their defaults.
func (r RetryConfig) Normalized() RetryConfig {
	if r.MaxAttempts < 2 {
		r.MaxAttempts = 3
	}
	if r.InitialBackoff <= 0 {
		r.InitialBackoff = 100 * time.Millisecond
	}
	if r.MaxBackoff <= 0 {
		r.MaxBackoff = time.Second
	}
	if r.BackoffMultiplier <= 0 {
		r.BackoffMultiplier = 2.0
	}
	return r
}

// Option configures a protocol client. It is the shared contract every
// protocol's NewClient accepts; protocol-specific dial knobs stay inside each
// protocol package.
type Option func(*ClientOptions)

// ClientOptions carries the generic knobs shared by every protocol client: the
// middleware chain and the retry policy.
type ClientOptions struct {
	// Middlewares is the outbound middleware chain, outermost first. Each
	// protocol converts it to its native chain before use.
	Middlewares []middleware.Middleware
	Retry       *RetryConfig
}

// Apply folds opts into o.
func (o *ClientOptions) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithMiddleware appends middleware to the client's outbound chain. Middleware
// added first runs first (outermost). It is protocol-agnostic: each protocol
// translates the chain to its native extension point (grpc unary interceptor,
// http RoundTripper, rpcx call wrapper, websocket dial).
func WithMiddleware(ms ...middleware.Middleware) Option {
	return func(o *ClientOptions) { o.Middlewares = append(o.Middlewares, ms...) }
}

// WithRetry enables retry for the client, translated to the protocol's native
// retry mechanism.
func WithRetry(r RetryConfig) Option {
	return func(o *ClientOptions) { o.Retry = &r }
}

// Client is the outbound-client contract a protocol transport package
// implements. NewClient builds a protocol-native handle (e.g. *grpc.ClientConn,
// *http.Client, WSDialer) from an address and the shared options; the concrete
// returned value is protocol-specific. transport/rpcx exposes its own typed
// constructor instead, because it needs a service name plus a list of addresses.
type Client interface {
	NewClient(addr string, opts ...Option) (any, error)
}
