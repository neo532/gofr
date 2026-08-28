package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
)

var _ client.Client = (*clientImpl)(nil)

// clientImpl implements client.Client for http.
type clientImpl struct{}

// NewClient builds an *http.Client whose transport layers the configured
// middleware chain over a pooled http.Transport, with the retry policy as an
// intermediate RoundTripper. The returned client is passed to the generated
// typed client (NewXxxHTTPClient), which supplies the base URL.
func NewClient(opts ...client.Option) *http.Client {
	o := &client.ClientOptions{}
	o.Apply(opts...)

	var tr http.RoundTripper = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
	if o.Retry != nil {
		tr = newRetryTripper(o.Retry.Normalized(), tr)
	}
	if len(o.Middlewares) > 0 {
		tr = httpTripper(middleware.Chain(o.Middlewares...), tr)
	}
	return &http.Client{Transport: tr}
}

func (clientImpl) NewClient(_ string, opts ...client.Option) (any, error) {
	return NewClient(opts...), nil
}

// roundTripperFunc adapts a func to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// httpTripper adapts a gofr middleware chain to an http.RoundTripper. The
// outbound header carrier wraps the request's own headers, so middleware
// mutations land directly on the wire. The request context is upgraded to the
// middleware context so spans and deadlines flow to the actual call.
func httpTripper(chain middleware.Middleware, base http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		ctx := client.WithOutgoingHeader(req.Context(), headerCarrier(req.Header))
		core := func(ctx context.Context, _ any) (any, error) {
			return base.RoundTrip(req.WithContext(ctx))
		}
		out, err := chain(core)(ctx, req)
		resp, _ := out.(*http.Response)
		return resp, err
	})
}

// retryTripper retries transport errors and retryable status codes (5xx by
// default, plus any codes listed in RetryConfig.Retryable) with exponential
// backoff. The request body is buffered so it can be replayed across attempts.
type retryTripper struct {
	base  http.RoundTripper
	retry client.RetryConfig
	codes map[int]bool
}

func newRetryTripper(r client.RetryConfig, base http.RoundTripper) *retryTripper {
	codes := make(map[int]bool, len(r.Retryable))
	for _, s := range r.Retryable {
		if n, err := strconv.Atoi(s); err == nil {
			codes[n] = true
		}
	}
	return &retryTripper{base: base, retry: r, codes: codes}
}

func (t *retryTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
	}
	backoff := t.retry.InitialBackoff
	var lastErr error
	for attempt := 1; ; attempt++ {
		if attempt > 1 && req.Body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err := t.base.RoundTrip(req)
		if err != nil {
			lastErr = err
		} else {
			if retryable := resp.StatusCode >= http.StatusInternalServerError || t.codes[resp.StatusCode]; !retryable {
				return resp, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		if attempt >= t.retry.MaxAttempts {
			return nil, lastErr
		}
		time.Sleep(backoff)
		backoff = time.Duration(float64(backoff) * t.retry.BackoffMultiplier)
		if backoff > t.retry.MaxBackoff {
			backoff = t.retry.MaxBackoff
		}
	}
}
