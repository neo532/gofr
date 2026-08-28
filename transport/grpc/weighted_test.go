package grpc

import (
	"strings"
	"testing"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"

	"github.com/neo532/gofr/transport/client"
)

func TestWeightedServiceConfig(t *testing.T) {
	cfg, err := weightedServiceConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg, `"loadBalancingPolicy":"gofr_weighted"`) {
		t.Fatalf("config missing loadBalancingPolicy: %q", cfg)
	}
	if strings.Contains(cfg, "methodConfig") {
		t.Fatalf("config unexpectedly has methodConfig: %q", cfg)
	}

	retry := client.RetryConfig{MaxAttempts: 3}
	cfg, err = weightedServiceConfig(&retry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg, `"loadBalancingPolicy":"gofr_weighted"`) ||
		!strings.Contains(cfg, `"methodConfig":[{`) {
		t.Fatalf("config missing policy or retry: %q", cfg)
	}
}

// stubSubConn satisfies balancer.SubConn via its embedded nil interface; id
// distinguishes instances so they are distinct map keys.
type stubSubConn struct {
	balancer.SubConn
	id int
}

func TestWeightedPickerBuilderWeights(t *testing.T) {
	pb := &weightedPickerBuilder{}
	p := pb.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			stubSubConn{id: 1}: {Address: resolver.Address{Addr: "a", Attributes: attributes.New(weightAttrKey{}, 1)}},
			stubSubConn{id: 2}: {Address: resolver.Address{Addr: "b", Attributes: attributes.New(weightAttrKey{}, 3)}},
			stubSubConn{id: 3}: {Address: resolver.Address{Addr: "c"}}, // missing weight -> default 1
		},
	})
	if _, ok := p.(*weightedPicker); !ok {
		t.Fatalf("expected *weightedPicker, got %T", p)
	}
}

func TestWeightedPickerDistribution(t *testing.T) {
	sa := stubSubConn{id: 1}
	sb := stubSubConn{id: 2}
	sc := stubSubConn{id: 3}
	p := &weightedPicker{scs: []scWeight{
		{sc: sa, weight: 1},
		{sc: sb, weight: 3},
		{sc: sc, weight: 6},
	}}

	const n = 20000
	counts := map[balancer.SubConn]int{}
	for range n {
		got, err := p.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatal(err)
		}
		counts[got.SubConn]++
	}
	want := []struct {
		sc   balancer.SubConn
		frac float64
	}{{sa, 0.1}, {sb, 0.3}, {sc, 0.6}}
	for _, w := range want {
		got := float64(counts[w.sc]) / n
		if got < w.frac-0.02 || got > w.frac+0.02 {
			t.Fatalf("subconn fraction %.2f, want ~%.2f (counts=%v)", got, w.frac, counts)
		}
	}
}
