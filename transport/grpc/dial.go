package grpc

import (
	"context"
	"net/url"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gofr/transport/client"
	"github.com/neo532/gokit/errorx"
)

// Dial builds a weighted grpc client from a registry instance set: it parses
// the grpc endpoints out of the instances, dials them through
// NewClientWithEndpoints, and returns the *grpc.ClientConn as the handle. Wire
// it as the GRPC dialer of a client.Dialers so a FollowClient re-dials on
// instance-set changes.
func Dial(opts ...client.Option) client.DialFunc {
	return func(_ context.Context, instances []*registry.ServiceInstance) (any, string, func(), error) {
		endpoints := grpcEndpoints(instances)
		if len(endpoints) == 0 {
			return nil, "", nil, errorx.New("grpc: no grpc endpoint in %+v", instances)
		}
		conn, err := NewClientWithEndpoints(endpoints, opts...)
		if err != nil {
			return nil, "", nil, errorx.Wrap(err)
		}
		return conn, "", func() { _ = conn.Close() }, nil
	}
}

// grpcEndpoints extracts the weighted grpc backends from an instance set.
func grpcEndpoints(instances []*registry.ServiceInstance) []Endpoint {
	out := make([]Endpoint, 0, len(instances))
	for _, inst := range instances {
		for _, raw := range inst.Endpoints {
			u, err := url.Parse(raw)
			if err != nil || u.Scheme != "grpc" || u.Host == "" {
				continue
			}
			out = append(out, Endpoint{Addr: u.Host, Weight: inst.Weight})
			break
		}
	}
	return out
}
