package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/neo532/gofr/registry"
)

const (
	keyPrefix  = "registry/"
	defaultTTL = 10 * time.Second
)

// Registrar stores each instance as registry/{name}/{id} under an etcd lease.
// KeepAlive renews the lease until the instance is Deregistered or the
// process crashes (lease then expires and etcd removes the key itself).
type Registrar struct {
	client *clientv3.Client
	keyPrefix string
	ttl    time.Duration

	mu     sync.Mutex
	leases map[string]leaseInfo
	closed bool
}

type leaseInfo struct {
	id     clientv3.LeaseID
	cancel context.CancelFunc
}

func NewRegistrar(endpoints []string, opts ...Option) (*Registrar, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	r := &Registrar{
		client:    cli,
		keyPrefix: keyPrefix,
		ttl:       defaultTTL,
		leases:    make(map[string]leaseInfo),
	}
	for _, o := range opts {
		o(r)
	}
	return r, nil
}

// key builds the storage key for an instance: registry/{group}/{name}/{id}.
func (r *Registrar) key(inst *registry.ServiceInstance) string {
	group := inst.Group
	if group == "" {
		group = registry.DefaultGroup
	}
	return r.keyPrefix + group + "/" + inst.Name + "/" + inst.ID
}

func (r *Registrar) Register(ctx context.Context, instance *registry.ServiceInstance) error {
	if instance == nil || instance.ID == "" || instance.Name == "" {
		return fmt.Errorf("registry: instance ID and Name are required")
	}
	key := r.key(instance)
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	leaseResp, err := r.client.Grant(ctx, int64(r.ttl.Seconds()))
	if err != nil {
		return err
	}
	if _, err := r.client.Put(ctx, key, string(data), clientv3.WithLease(leaseResp.ID)); err != nil {
		r.client.Revoke(context.Background(), leaseResp.ID)
		return err
	}

	kaCtx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		cancel()
		r.client.Delete(context.Background(), key)
		r.client.Revoke(context.Background(), leaseResp.ID)
		return fmt.Errorf("registry: registrar closed")
	}
	if prev, ok := r.leases[key]; ok {
		prev.cancel() // replace: only one keepalive per instance
	}
	if _, err := r.client.KeepAlive(kaCtx, leaseResp.ID); err != nil {
		cancel()
		r.client.Revoke(context.Background(), leaseResp.ID)
		return err
	}
	r.leases[key] = leaseInfo{id: leaseResp.ID, cancel: cancel}
	return nil
}

func (r *Registrar) Deregister(ctx context.Context, instance *registry.ServiceInstance) error {
	key := r.key(instance)
	r.mu.Lock()
	info, ok := r.leases[key]
	if ok {
		delete(r.leases, key)
	}
	r.mu.Unlock()
	if ok {
		info.cancel()
		r.client.Revoke(context.Background(), info.id)
	}
	_, err := r.client.Delete(ctx, key)
	return err
}

func (r *Registrar) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	for key, info := range r.leases {
		info.cancel()
		delete(r.leases, key)
		r.client.Revoke(context.Background(), info.id)
	}
	// Remaining keys expire with their leases; the App Deregisters before Close.
	return r.client.Close()
}
