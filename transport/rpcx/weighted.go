package rpcx

import (
	"strconv"
	"time"

	rpcxClient "github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
)

// Endpoint is a single backend with an explicit load-balancing weight. Weight 0
// or negative falls back to 1.
type Endpoint struct {
	Addr   string
	Weight int
}

// NewClientWithEndpoints builds a weighted rpcx client for service against the
// given endpoints. It is NewClient with per-endpoint weights: each backend's
// Weight becomes the "weight=" value of its discovery KVPair, which rpcx's
// WeightedRoundRobin selector consumes natively. Middleware, retry and the
// outbound header carrier behave exactly as in NewClient.
func NewClientWithEndpoints(service string, endpoints []Endpoint, opts ...client.Option) (rpcxClient.XClient, error) {
	o := &client.ClientOptions{}
	o.Apply(opts...)

	pairs := make([]*rpcxClient.KVPair, 0, len(endpoints))
	for _, ep := range endpoints {
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		pairs = append(pairs, &rpcxClient.KVPair{Key: "tcp@" + ep.Addr, Value: "weight=" + strconv.Itoa(weight)})
	}
	d, err := rpcxClient.NewMultipleServersDiscovery(pairs)
	if err != nil {
		return nil, err
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
