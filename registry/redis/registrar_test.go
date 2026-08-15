package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/neo532/gofr/registry"
)

func testClient(t *testing.T) *redis.Client {
	t.Helper()
	c := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := c.Ping(context.Background()).Err(); err != nil {
		c.Close()
		t.Skipf("redis not available at 127.0.0.1:6379: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// testPrefix returns an isolated prefix; keys under it are cleaned up on exit.
func testPrefix(t *testing.T) (string, string) {
	t.Helper()
	key := fmt.Sprintf("registry:test:%d:", time.Now().UnixNano())
	chanp := fmt.Sprintf("registry:testchan:%d:", time.Now().UnixNano())
	c := testClient(t)
	t.Cleanup(func() {
		iter := c.Scan(context.Background(), 0, key+"*", 100).Iterator()
		for iter.Next(context.Background()) {
			c.Del(context.Background(), iter.Val())
		}
	})
	return key, chanp
}

func instance(id, name string) *registry.ServiceInstance {
	return &registry.ServiceInstance{ID: id, Name: name, Endpoints: []string{"rpcx://10.0.0.1:8503"}}
}

func TestRegistrarGetService(t *testing.T) {
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(2*time.Second))
	defer reg.Close()
	dis := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
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
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(2*time.Second))
	defer reg.Close()
	dis := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
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

// TestRegistrarHeartbeat: the lease is renewed by the heartbeat, so the
// instance survives past the ttl while the registrar is alive.
func TestRegistrarHeartbeat(t *testing.T) {
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(1*time.Second))
	defer reg.Close()
	dis := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
	defer dis.Close()

	if err := reg.Register(context.Background(), instance("ip1", "user")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond) // > ttl, heartbeat must have renewed
	got, err := dis.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want instance still alive after ttl, got %+v", got)
	}
}

// TestRegistrarCrashExpiry: when the registrar dies without Deregister
// (heartbeat stops), redis expires the key after the ttl.
func TestRegistrarCrashExpiry(t *testing.T) {
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(1*time.Second))
	if err := reg.Register(context.Background(), instance("ip1", "user")); err != nil {
		t.Fatal(err)
	}
	reg.Close() // heartbeat stops; no Deregister, as on a crash
	dis := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
	defer dis.Close()

	time.Sleep(1500 * time.Millisecond) // wait for the lease to lapse
	got, err := dis.GetService(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want instance expired, got %+v", got)
	}
}

func TestRegistrarDeregister(t *testing.T) {
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(2*time.Second))
	defer reg.Close()
	dis := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
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
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(3*time.Second))
	defer reg.Close()
	dis := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
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

	// Read until a snapshot containing ip2 arrives (first Next may carry the
	// pre-registration state via the poll fallback).
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
	key, chanp := testPrefix(t)
	reg := NewRegistrar("127.0.0.1:6379", WithPrefix(key, chanp), WithTTL(2*time.Second))
	defer reg.Close()
	disDefault := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp))
	defer disDefault.Close()
	disNeo := NewDiscovery("127.0.0.1:6379", WithPrefix(key, chanp), WithGroup("neo"))
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
