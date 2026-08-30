package grpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
	"github.com/neo532/gokit/errorx"
)

var _ client.Client = (*Client)(nil)

// Client implements client.Client for grpc: NewClient dials a backend at
// addr through the package-level NewClient (middleware + retry).
type Client struct{}

// NewClient dials a grpc backend at addr with the configured middleware chain
// and retry policy. The returned *grpc.ClientConn is passed to the generated
// typed client (NewXxxGRPCClient).
//
// The middleware chain runs once per RPC: each call installs a fresh outbound
// header carrier (client.OutgoingHeader), runs the chain, then merges the
// carrier into the outgoing metadata before invoking. grpc's retryPolicy
// retries below the interceptor, so middleware does not re-run on a retry.
// Credentials default to insecure; TLS is not configurable yet.
func NewClient(addr string, opts ...client.Option) (*grpc.ClientConn, error) {
	o := &client.ClientOptions{}
	o.Apply(opts...)

	dial := make([]grpc.DialOption, 0, 2)
	dial = append(dial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if len(o.Middlewares) > 0 {
		chain := middleware.Chain(o.Middlewares...)
		dial = append(dial, grpc.WithUnaryInterceptor(grpcInterceptor(chain)))
	}
	if o.Retry != nil {
		cfg, err := grpcServiceConfig(*o.Retry)
		if err != nil {
			return nil, errorx.Wrap(err)
		}
		dial = append(dial, grpc.WithDefaultServiceConfig(cfg))
	}
	return grpc.NewClient(addr, dial...)
}

func (Client) NewClient(addr string, opts ...client.Option) (any, error) {
	return NewClient(addr, opts...)
}

// grpcInterceptor adapts a gofr middleware chain to a grpc unary client
// interceptor. The terminal handler closes over reply, so it is rebuilt per
// call.
func grpcInterceptor(chain middleware.Middleware) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		carrier := headerCarrier(metadata.MD{})
		ctx = client.WithOutgoingHeader(ctx, carrier)
		core := func(ctx context.Context, r any) (any, error) {
			out := metadata.Join(metadata.MD(carrier))
			if existing, ok := metadata.FromOutgoingContext(ctx); ok {
				out = metadata.Join(existing, out)
			}
			ic := metadata.NewOutgoingContext(ctx, out)
			err := invoker(ic, method, r, reply, cc, opts...)
			return reply, errorx.Wrap(err)
		}
		_, err := chain(core)(ctx, req)
		return errorx.Wrap(err)
	}
}

// defaultGRPCRetryable are the proto status-code names grpc retries by default.
var defaultGRPCRetryable = []string{
	"UNAVAILABLE", "DEADLINE_EXCEEDED", "RESOURCE_EXHAUSTED", "ABORTED", "INTERNAL", "DATA_LOSS",
}

// grpcServiceConfig renders RetryConfig as the grpc retryPolicy service config.
// Durations are protobuf "Xs" strings; status codes are the uppercase proto
// names grpc-go's strToCode map accepts.
func grpcServiceConfig(r client.RetryConfig) (string, error) {
	r = r.Normalized()
	retryable := r.Retryable
	if len(retryable) == 0 {
		retryable = defaultGRPCRetryable
	}
	names := make([]string, 0, len(retryable))
	for _, n := range retryable {
		names = append(names, fmt.Sprintf("%q", strings.ToUpper(n)))
	}
	return fmt.Sprintf(`{"methodConfig":[{"name":[{"service":""}],"retryPolicy":{`+
		`"maxAttempts":%d,`+
		`"initialBackoff":%q,`+
		`"maxBackoff":%q,`+
		`"backoffMultiplier":%g,`+
		`"retryableStatusCodes":[%s]`+
		`}}]}`, r.MaxAttempts, durationString(r.InitialBackoff), durationString(r.MaxBackoff), r.BackoffMultiplier, strings.Join(names, ",")), nil
}

// durationString renders a duration in the protobuf "Xs" format grpc's service
// config parser expects (e.g. 100ms -> "0.1s").
func durationString(d time.Duration) string {
	return fmt.Sprintf("%gs", d.Seconds())
}
