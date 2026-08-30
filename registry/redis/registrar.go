package redis

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gokit/errorx"
)

const (
	keyPrefix  = "registry:svc:"
	chanPrefix = "registry:chan:"
	defaultTTL = 10 * time.Second
)

// Registrar registers service instances as per-instance keys with a real
// redis TTL lease: each instance is one key (registry:svc:{name}:{id}), so an
// expired lease removes only the crashed instance, not the whole service. A
// shared hash with one TTL would evict every instance when any single lease
// lapses, which is wrong for the multi-instance topology this registry exists
// for.
type Registrar struct {
	client *redis.Client
	cfg    config

	mu         sync.Mutex
	registered map[string]context.CancelFunc
	closed     bool
}

func NewRegistrar(addr string, opts ...Option) *Registrar {
	r := &Registrar{
		client:     redis.NewClient(&redis.Options{Addr: addr, PoolSize: 20}),
		cfg:        defaultConfig(),
		registered: make(map[string]context.CancelFunc),
	}
	for _, o := range opts {
		o(&r.cfg)
	}
	return r
}

// groupOf returns the group an instance registers under, defaulting to
// registry.DefaultGroup.
func (r *Registrar) groupOf(inst *registry.ServiceInstance) string {
	if inst.Group == "" {
		return registry.DefaultGroup
	}
	return inst.Group
}

func (r *Registrar) Register(ctx context.Context, instance *registry.ServiceInstance) error {
	if instance == nil || instance.ID == "" || instance.Name == "" {
		return errorx.New("registry: instance ID and Name are required")
	}
	group := r.groupOf(instance)
	key := r.cfg.keyPrefix + group + ":" + instance.Name + ":" + instance.ID
	data, err := json.Marshal(instance)
	if err != nil {
		return errorx.Wrap(err)
	}
	if err := r.client.Set(ctx, key, data, r.cfg.ttl).Err(); err != nil {
		return errorx.Wrap(err)
	}
	r.publish(ctx, group, instance.Name)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		r.client.Del(context.Background(), key)
		return errorx.New("registry: registrar closed")
	}
	if _, ok := r.registered[key]; ok {
		return nil // heartbeat already running for this instance
	}
	hbCtx, cancel := context.WithCancel(context.Background())
	r.registered[key] = cancel
	go r.heartbeat(hbCtx, key)
	return nil
}

func (r *Registrar) Deregister(ctx context.Context, instance *registry.ServiceInstance) error {
	group := r.groupOf(instance)
	key := r.cfg.keyPrefix + group + ":" + instance.Name + ":" + instance.ID
	r.mu.Lock()
	if cancel, ok := r.registered[key]; ok {
		cancel()
		delete(r.registered, key)
	}
	r.mu.Unlock()
	// EXPIRE on a missing key is a no-op, so a heartbeat tick racing this DEL
	// cannot resurrect the key.
	err := r.client.Del(ctx, key).Err()
	if err == nil {
		r.publish(ctx, group, instance.Name)
	}
	return errorx.Wrap(err)
}

// Check verifies the registry is reachable (fail-fast startup gate).
func (r *Registrar) Check(ctx context.Context) error {
	return errorx.Wrap(r.client.Ping(ctx).Err())
}

func (r *Registrar) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	for key, cancel := range r.registered {
		cancel()
		delete(r.registered, key)
	}
	// Remaining keys expire on their own TTL; the App Deregisters before Close.
	return errorx.Wrap(r.client.Close())
}

// heartbeat renews the key lease until cancelled. It runs on its own context:
// Register's request-scoped ctx must not be used here, the lease must outlive
// the Register call.
func (r *Registrar) heartbeat(ctx context.Context, key string) {
	t := time.NewTicker(r.cfg.ttl / 3)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Transient redis errors are retried on the next tick.
			r.client.Expire(ctx, key, r.cfg.ttl)
		}
	}
}

// publish is called on every mutation so watchers get a fast-path notification.
func (r *Registrar) publish(ctx context.Context, group, name string) {
	r.client.Publish(ctx, r.cfg.chanPrefix+group+":"+name, "1")
}
