package rpcx

import (
	"context"
	"maps"
	"time"

	rpcxClient "github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
	"github.com/neo532/gokit/errorx"
)

var _ client.Client = (*Client)(nil)

// Client implements client.Client for rpcx. rpcx's native NewClient needs a
// service name plus a list of addresses, which the client.Client contract
// cannot express in a single addr, so the Client carries both and NewClient
// ignores its addr argument.
type Client struct {
	Service string
	Addrs   []string
}

func (c *Client) NewClient(addr string, opts ...client.Option) (any, error) {
	return NewClient(c.Service, c.Addrs, opts...)
}

// NewClient builds a rpcx client for service against the discovered addresses.
// It returns a client.XClient whose Call runs the middleware chain and injects
// the outbound header carrier as rpcx request metadata, retrying via rpcx's
// native Failtry mode. The returned value is passed to the generated typed
// client (NewXxxRPCXClient).
//
// rpcx's Failtry retries are coarse (a retry count, not per-condition), so
// RetryConfig.Retryable is ignored; only MaxAttempts drives opt.Retries. The
// (service, addrs) signature diverges from client.Client, so rpcx does not
// implement that interface.
func NewClient(service string, addrs []string, opts ...client.Option) (rpcxClient.XClient, error) {
	o := &client.ClientOptions{}
	o.Apply(opts...)

	pairs := make([]*rpcxClient.KVPair, 0, len(addrs))
	for _, a := range addrs {
		pairs = append(pairs, &rpcxClient.KVPair{Key: "tcp@" + a, Value: "weight=1"})
	}
	d, err := rpcxClient.NewMultipleServersDiscovery(pairs)
	if err != nil {
		return nil, errorx.Wrap(err)
	}
	opt := rpcxClient.DefaultOption
	opt.SerializeType = protocol.ProtoBuffer
	// rpcx defaults Retries to 3; gofr's default is no retry (matching grpc/http/
	// websocket), so zero it and retry only when WithRetry is set.
	opt.Retries = 0
	if o.Retry != nil {
		opt.Retries = o.Retry.Normalized().MaxAttempts - 1
	}
	// The server closes any connection whose rpcx read timeout expires with no
	// traffic, and the client pool would otherwise hold a silently-dead
	// connection. Heartbeat pings reset the server's read deadline, so the
	// interval must stay well under the server's rpcx read timeout.
	opt.Heartbeat = true
	opt.HeartbeatInterval = time.Second
	real := rpcxClient.NewXClient(service, rpcxClient.Failtry, rpcxClient.WeightedRoundRobin, d, opt)
	if len(o.Middlewares) == 0 {
		return real, nil
	}
	return &rpcxWrapped{XClient: real, chain: middleware.Chain(o.Middlewares...)}, nil
}

// rpcxWrapped embeds the real XClient so every non-Call method delegates, and
// intercepts Call to run the middleware chain and forward the outbound header
// carrier as rpcx request metadata.
type rpcxWrapped struct {
	rpcxClient.XClient
	chain middleware.Middleware
}

func (w *rpcxWrapped) Call(ctx context.Context, serviceMethod string, args, reply any) error {
	meta := make(map[string]string)
	// Preserve metadata the caller injected on ctx (e.g. share.ReqMetaDataKey),
	// mirroring grpc, which merges metadata.FromOutgoingContext.
	if pre, ok := ctx.Value(share.ReqMetaDataKey).(map[string]string); ok {
		maps.Copy(meta, pre)
	}
	ctx = client.WithOutgoingHeader(ctx, headerCarrier(meta))
	core := func(ctx context.Context, r any) (any, error) {
		c := context.WithValue(ctx, share.ReqMetaDataKey, meta)
		err := w.XClient.Call(c, serviceMethod, r, reply)
		return reply, errorx.Wrap(err)
	}
	_, err := w.chain(core)(ctx, args)
	return errorx.Wrap(err)
}
