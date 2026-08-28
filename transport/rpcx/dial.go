package rpcx

import (
	"context"
	"fmt"
	"net/url"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gofr/transport/client"
	rpcxClient "github.com/smallnest/rpcx/client"
)

// Dial builds one XClient per rpcx service in serviceNames, all over the same
// weighted backends, from a registry instance set. rpcx binds an XClient to a
// single service name, so the handle is a map the generated typed clients key
// by their own service name. Wire it as the RPCX dialer of a client.Dialers so
// a FollowClient re-dials on instance-set changes.
func Dial(serviceNames []string, opts ...client.Option) client.DialFunc {
	return func(_ context.Context, instances []*registry.ServiceInstance) (any, string, func(), error) {
		endpoints := rpcxEndpoints(instances)
		if len(endpoints) == 0 {
			return nil, "", nil, fmt.Errorf("rpcx: no rpcx endpoint in %+v", instances)
		}
		xcs := make(map[string]rpcxClient.XClient, len(serviceNames))
		closeAll := func() {
			for _, xc := range xcs {
				_ = xc.Close()
			}
		}
		for _, svcName := range serviceNames {
			xc, err := NewClientWithEndpoints(svcName, endpoints, opts...)
			if err != nil {
				closeAll()
				return nil, "", nil, err
			}
			xcs[svcName] = xc
		}
		return xcs, "", closeAll, nil
	}
}

// rpcxEndpoints extracts the weighted rpcx backends from an instance set.
func rpcxEndpoints(instances []*registry.ServiceInstance) []Endpoint {
	out := make([]Endpoint, 0, len(instances))
	for _, inst := range instances {
		for _, raw := range inst.Endpoints {
			u, err := url.Parse(raw)
			if err != nil || u.Scheme != "rpcx" || u.Host == "" {
				continue
			}
			out = append(out, Endpoint{Addr: u.Host, Weight: inst.Weight})
			break
		}
	}
	return out
}
