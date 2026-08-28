package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
)

var _ client.Client = (*clientImpl)(nil)

// clientImpl implements client.Client for websocket.
type clientImpl struct{}

// WSDialer dials a websocket endpoint. *websocket.Dialer satisfies it, and the
// generated typed client (NewXxxWSClient) accepts any value that does, so the
// wrapper returned by NewClient plugs straight in.
type WSDialer interface {
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

// NewClient builds a websocket dialer with the middleware chain applied to the
// upgrade-handshake headers and retry on dial failure. Since each generated
// websocket call opens a fresh connection, retry here only covers dialing;
// per-call (dial + write + read) retry stays at the application layer.
func NewClient(opts ...client.Option) WSDialer {
	o := &client.ClientOptions{}
	o.Apply(opts...)

	var d WSDialer = &websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	if o.Retry != nil {
		d = &wsRetryDialer{Dialer: d, retry: o.Retry.Normalized()}
	}
	if len(o.Middlewares) > 0 {
		d = &wsMetaDialer{Dialer: d, chain: middleware.Chain(o.Middlewares...)}
	}
	return d
}

func (clientImpl) NewClient(_ string, opts ...client.Option) (any, error) {
	return NewClient(opts...), nil
}

// wsMetaDialer runs the middleware chain once per dial, exposing the upgrade
// request headers through OutgoingHeader. Retry (when present) is below it, so
// middleware runs once even across dial retries — mirroring grpc, where the
// interceptor runs once and retryPolicy retries below.
type wsMetaDialer struct {
	Dialer WSDialer
	chain  middleware.Middleware
}

func (d *wsMetaDialer) DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	if requestHeader == nil {
		requestHeader = http.Header{}
	}
	ctx = client.WithOutgoingHeader(ctx, headerCarrier(requestHeader))
	var conn *websocket.Conn
	var resp *http.Response
	core := func(ctx context.Context, _ any) (any, error) {
		var err error
		conn, resp, err = d.Dialer.DialContext(ctx, urlStr, requestHeader)
		return conn, err
	}
	_, err := d.chain(core)(ctx, urlStr)
	return conn, resp, err
}

// wsRetryDialer retries a failed dial with exponential backoff.
type wsRetryDialer struct {
	Dialer WSDialer
	retry  client.RetryConfig
}

func (d *wsRetryDialer) DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	backoff := d.retry.InitialBackoff
	var lastErr error
	for attempt := 1; ; attempt++ {
		conn, resp, err := d.Dialer.DialContext(ctx, urlStr, requestHeader)
		if err == nil {
			return conn, resp, nil
		}
		lastErr = err
		if attempt >= d.retry.MaxAttempts {
			return nil, nil, lastErr
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = time.Duration(float64(backoff) * d.retry.BackoffMultiplier)
		if backoff > d.retry.MaxBackoff {
			backoff = d.retry.MaxBackoff
		}
	}
}
