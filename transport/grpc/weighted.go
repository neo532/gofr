package grpc

import (
	"math/rand/v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport/client"
)

// gofrWeightScheme is the one-off resolver scheme used by
// NewClientWithEndpoints. The scheme is never dialed by name; grpc.NewClient
// receives "gofrweight:///" only so it routes resolution to the embedded
// endpoints resolver below.
const gofrWeightScheme = "gofrweight"

// gofrWeightedPolicy is the load-balancing policy name registered at init and
// selected through the client's service config.
const gofrWeightedPolicy = "gofr_weighted"

// weightAttrKey carries a backend's weight on resolver.Address.Attributes. It is
// a private package type so it cannot collide with grpc-go's own attribute keys.
type weightAttrKey struct{}

// Endpoint is a single backend with an explicit load-balancing weight. Weight 0
// or negative falls back to 1.
type Endpoint struct {
	Addr   string
	Weight int
}

func init() {
	balancer.Register(base.NewBalancerBuilder(gofrWeightedPolicy, &weightedPickerBuilder{}, base.Config{HealthCheck: false}))
}

// NewClientWithEndpoints dials a weighted multi-backend grpc client, layered on
// the same middleware and retry knobs as NewClient. Each Endpoint is reported to
// a custom resolver under the gofrweight scheme carrying its weight, and the
// gofr_weighted policy picks backends by weighted random selection. Unlike
// grpc-go's weighted_round_robin (load-report driven), weights here are the
// static service weights registered in the discovery registry, so no load
// reporting is involved.
//
// The returned *grpc.ClientConn is passed to the generated typed client
// (NewXxxGRPCClient), exactly like NewClient's.
func NewClientWithEndpoints(endpoints []Endpoint, opts ...client.Option) (*grpc.ClientConn, error) {
	o := &client.ClientOptions{}
	o.Apply(opts...)

	dial := make([]grpc.DialOption, 0, 4)
	dial = append(dial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dial = append(dial, grpc.WithResolvers(&endpointsResolverBuilder{endpoints: endpoints}))

	cfg, err := weightedServiceConfig(o.Retry)
	if err != nil {
		return nil, err
	}
	dial = append(dial, grpc.WithDefaultServiceConfig(cfg))

	if len(o.Middlewares) > 0 {
		chain := middleware.Chain(o.Middlewares...)
		dial = append(dial, grpc.WithUnaryInterceptor(grpcInterceptor(chain)))
	}
	return grpc.NewClient(gofrWeightScheme+":///", dial...)
}

// weightedServiceConfig renders the service config for the gofr_weighted policy,
// merging the retry methodConfig (if configured) with the load-balancing policy.
func weightedServiceConfig(retry *client.RetryConfig) (string, error) {
	if retry == nil {
		return `{"loadBalancingPolicy":"` + gofrWeightedPolicy + `"}`, nil
	}
	rc, err := grpcServiceConfig(*retry)
	if err != nil {
		return "", err
	}
	// rc is {"methodConfig":[...]} — splice the load-balancing policy in.
	return `{"loadBalancingPolicy":"` + gofrWeightedPolicy + `",` + rc[1:], nil
}

// endpointsResolverBuilder is a one-off resolver that reports the weighted
// endpoints exactly once, at Build time, since the address set is fixed for the
// lifetime of the connection.
type endpointsResolverBuilder struct {
	endpoints []Endpoint
}

func (b *endpointsResolverBuilder) Build(_ resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	addrs := make([]resolver.Address, 0, len(b.endpoints))
	for _, ep := range b.endpoints {
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		addrs = append(addrs, resolver.Address{
			Addr:       ep.Addr,
			Attributes: attributes.New(weightAttrKey{}, weight),
		})
	}
	if err := cc.UpdateState(resolver.State{Addresses: addrs}); err != nil {
		return nil, err
	}
	return &noopResolver{}, nil
}

func (b *endpointsResolverBuilder) Scheme() string { return gofrWeightScheme }

// noopResolver holds a connection idle after the initial update.
type noopResolver struct{}

func (noopResolver) ResolveNow(resolver.ResolveNowOptions) {}
func (noopResolver) Close()                                {}

// weightedPickerBuilder builds a weightedPicker from the ready SubConns.
type weightedPickerBuilder struct{}

func (*weightedPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	scs := make([]scWeight, 0, len(info.ReadySCs))
	for sc, si := range info.ReadySCs {
		weight, _ := si.Address.Attributes.Value(weightAttrKey{}).(int)
		if weight <= 0 {
			weight = 1
		}
		scs = append(scs, scWeight{sc: sc, weight: weight})
	}
	return &weightedPicker{scs: scs}
}

type scWeight struct {
	sc     balancer.SubConn
	weight int
}

// weightedPicker picks a SubConn by weighted random selection over the static
// weights carried on each address.
type weightedPicker struct {
	scs []scWeight
}

func (p *weightedPicker) Pick(_ balancer.PickInfo) (balancer.PickResult, error) {
	total := 0
	for _, s := range p.scs {
		total += s.weight
	}
	n := rand.IntN(total)
	for _, s := range p.scs {
		n -= s.weight
		if n < 0 {
			return balancer.PickResult{SubConn: s.sc}, nil
		}
	}
	// Unreachable: every weight is normalized to >= 1, so total > 0.
	return balancer.PickResult{SubConn: p.scs[0].sc}, nil
}
