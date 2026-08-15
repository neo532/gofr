package etcd

import (
	"context"
	"net/url"
	"testing"
	"time"

	"go.etcd.io/etcd/server/v3/embed"

	"github.com/neo532/gofr/registry"
)

// startEmbeddedEtcd boots an in-process etcd on ephemeral ports and returns
// its client endpoint. Ephemeral ports avoid collisions between tests.
func startEmbeddedEtcd(t *testing.T) string {
	t.Helper()
	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	cfg.LogLevel = "error"
	cfg.ListenClientUrls = []url.URL{{Scheme: "http", Host: "127.0.0.1:0"}}
	cfg.AdvertiseClientUrls = cfg.ListenClientUrls
	cfg.ListenPeerUrls = []url.URL{{Scheme: "http", Host: "127.0.0.1:0"}}
	cfg.AdvertisePeerUrls = cfg.ListenPeerUrls
	// The initial-cluster is only rebuilt from AdvertisePeerUrls for a
	// non-default name, so pin it to match the ephemeral peer URL.
	cfg.InitialCluster = "default=" + cfg.AdvertisePeerUrls[0].String()

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatalf("start embedded etcd: %v", err)
	}
	t.Cleanup(func() { e.Close() })
	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(15 * time.Second):
		t.Fatal("embedded etcd not ready in time")
	}
	return "http://" + e.Clients[0].Addr().String()
}

func instance(id, name string) *registry.ServiceInstance {
	return &registry.ServiceInstance{ID: id, Name: name, Endpoints: []string{"rpcx://10.0.0.1:8503"}}
}

func TestRegistrarGetService(t *testing.T) {
	addr := startEmbeddedEtcd(t)
	reg, err := NewRegistrar([]string{addr}, WithTTL(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()
	dis, err := NewDiscovery([]string{addr})
	if err != nil {
		t.Fatal(err)
	}
	defer dis.Close()

	if err := reg.Register(context.Background(), instance("ip1", "user")); err != nil {
		t.Fatal(err)
	}
	got, err := dis.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "ip1" {
		t.Fatalf("want 1 instance ip1, got %+v", got)
	}
}

func TestRegistrarMultiInstance(t *testing.T) {
	addr := startEmbeddedEtcd(t)
	reg, err := NewRegistrar([]string{addr}, WithTTL(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()
	dis, err := NewDiscovery([]string{addr})
	if err != nil {
		t.Fatal(err)
	}
	defer dis.Close()

	for _, id := range []string{"ip1", "ip2"} {
		if err := reg.Register(context.Background(), instance(id, "user")); err != nil {
			t.Fatal(err)
		}
	}
	got, err := dis.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 instances, got %+v", got)
	}
}

// TestRegistrarGracefulShutdown: Close stops keepalives and revokes leases,
// so instances disappear without waiting for the lease TTL.
func TestRegistrarGracefulShutdown(t *testing.T) {
	addr := startEmbeddedEtcd(t)
	reg, err := NewRegistrar([]string{addr}, WithTTL(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	dis, err := NewDiscovery([]string{addr})
	if err != nil {
		t.Fatal(err)
	}
	defer dis.Close()

	if err := reg.Register(context.Background(), instance("ip1", "user")); err != nil {
		t.Fatal(err)
	}
	reg.Close()
	got, err := dis.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want instance removed after Close, got %+v", got)
	}
}

func TestRegistrarDeregister(t *testing.T) {
	addr := startEmbeddedEtcd(t)
	reg, err := NewRegistrar([]string{addr}, WithTTL(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()
	dis, err := NewDiscovery([]string{addr})
	if err != nil {
		t.Fatal(err)
	}
	defer dis.Close()

	inst := instance("ip1", "user")
	if err := reg.Register(context.Background(), inst); err != nil {
		t.Fatal(err)
	}
	if err := reg.Deregister(context.Background(), inst); err != nil {
		t.Fatal(err)
	}
	got, err := dis.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want instance removed, got %+v", got)
	}
}

func TestWatchNotification(t *testing.T) {
	addr := startEmbeddedEtcd(t)
	reg, err := NewRegistrar([]string{addr}, WithTTL(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()
	dis, err := NewDiscovery([]string{addr})
	if err != nil {
		t.Fatal(err)
	}
	defer dis.Close()

	if err := reg.Register(context.Background(), instance("ip1", "user")); err != nil {
		t.Fatal(err)
	}
	w, err := dis.Watch(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := reg.Register(context.Background(), instance("ip2", "user")); err != nil {
		t.Fatal(err)
	}

	// First Next may carry the pre-registration snapshot; read until ip2 shows.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("watch never observed ip2")
		default:
		}
		instances, err := w.Next()
		if err != nil {
			t.Fatal(err)
		}
		if containsID(instances, "ip2") {
			return
		}
	}
}

// TestGroupResolution: a developer group overrides the default group for the
// services it has instances of, and falls back to default otherwise. Fields
// (group/protocol/weight) round-trip through the registry.
func TestGroupResolution(t *testing.T) {
	addr := startEmbeddedEtcd(t)
	reg, err := NewRegistrar([]string{addr}, WithTTL(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()
	disDefault, err := NewDiscovery([]string{addr})
	if err != nil {
		t.Fatal(err)
	}
	defer disDefault.Close()
	disNeo, err := NewDiscovery([]string{addr}, WithGroup("neo"))
	if err != nil {
		t.Fatal(err)
	}
	defer disNeo.Close()

	shared := instance("ip1", "user")
	if err := reg.Register(context.Background(), shared); err != nil {
		t.Fatal(err)
	}
	neo := instance("ip2", "user")
	neo.Group = "neo"
	neo.Protocol = "rpcx"
	neo.Weight = 100
	if err := reg.Register(context.Background(), neo); err != nil {
		t.Fatal(err)
	}

	got, err := disDefault.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "ip1" {
		t.Fatalf("default group: want shared ip1 only, got %+v", got)
	}

	got, err = disNeo.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "ip2" || got[0].Group != "neo" || got[0].Protocol != "rpcx" || got[0].Weight != 100 {
		t.Fatalf("neo group: want ip2 with fields, got %+v", got)
	}

	// Neo discovery falls back to default when its group is empty.
	if err := reg.Register(context.Background(), instance("ip3", "cms")); err != nil {
		t.Fatal(err)
	}
	got, err = disNeo.GetService(context.Background(), "cms")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "ip3" {
		t.Fatalf("neo fallback: want shared cms ip3, got %+v", got)
	}
}

func containsID(instances []*registry.ServiceInstance, id string) bool {
	for _, in := range instances {
		if in.ID == id {
			return true
		}
	}
	return false
}
