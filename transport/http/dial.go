package http

import (
	"context"
	"net/url"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gofr/transport/client"
	"github.com/neo532/gokit/errorx"
)

// Dial builds an *http.Client from a registry instance set, returning the first
// advertised http endpoint as the base URL the generated typed client prefixes
// its requests with. Wire it as the HTTP dialer of a client.Dialers so a
// FollowClient re-dials on instance-set changes.
func Dial(opts ...client.Option) client.DialFunc {
	return func(_ context.Context, instances []*registry.ServiceInstance) (any, string, func(), error) {
		baseURL, ok := firstBaseURL(instances, "http")
		if !ok {
			return nil, "", nil, errorx.New("http: no http endpoint in %+v", instances)
		}
		return NewClient(opts...), baseURL, func() {}, nil
	}
}

// firstBaseURL returns the first endpoint of the given scheme as a base URL
// (scheme://host), or "" if the instance set advertises none.
func firstBaseURL(instances []*registry.ServiceInstance, scheme string) (string, bool) {
	for _, inst := range instances {
		for _, raw := range inst.Endpoints {
			u, err := url.Parse(raw)
			if err != nil || u.Scheme != scheme || u.Host == "" {
				continue
			}
			return u.String(), true
		}
	}
	return "", false
}
