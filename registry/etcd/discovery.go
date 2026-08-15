package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/neo532/gofr/registry"
)

// pollInterval backs up etcd Watch events; the watch stream is authoritative,
// this only covers a missed or dropped change.
const pollInterval = 10 * time.Second

// Discovery resolves instances by preferred group, falling back to
// registry.DefaultGroup: for each service the first non-empty group in
// [group, default] wins, so a developer's local instance overrides the shared
// ones, and an absent local group falls back to the shared pool.
type Discovery struct {
	client *clientv3.Client
	group  string
}

func NewDiscovery(endpoints []string, opts ...DiscoveryOption) (*Discovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	d := &Discovery{client: cli}
	for _, o := range opts {
		o(d)
	}
	return d, nil
}

// groups returns the group resolution order: [preferred, default].
func (d *Discovery) groups() []string {
	if d.group == "" || d.group == registry.DefaultGroup {
		return []string{registry.DefaultGroup}
	}
	return []string{d.group, registry.DefaultGroup}
}

func (d *Discovery) prefix(group, name string) string {
	return keyPrefix + group + "/" + name + "/"
}

func (d *Discovery) GetService(ctx context.Context, name string) ([]*registry.ServiceInstance, error) {
	for _, g := range d.groups() {
		instances, err := d.getInGroup(ctx, g, name)
		if err != nil {
			return nil, err
		}
		if len(instances) > 0 {
			return instances, nil
		}
	}
	return nil, nil
}

func (d *Discovery) getInGroup(ctx context.Context, group, name string) ([]*registry.ServiceInstance, error) {
	resp, err := d.client.Get(ctx, d.prefix(group, name), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var instances []*registry.ServiceInstance
	for _, kv := range resp.Kvs {
		var inst registry.ServiceInstance
		if err := json.Unmarshal(kv.Value, &inst); err != nil {
			return nil, err
		}
		instances = append(instances, &inst)
	}
	return instances, nil
}

func (d *Discovery) Watch(ctx context.Context, name string) (registry.Watcher, error) {
	wCtx, cancel := context.WithCancel(ctx)
	w := &watcher{
		d:      d,
		name:   name,
		ctx:    wCtx,
		cancel: cancel,
		ch:     make(chan []*registry.ServiceInstance, 1),
	}
	go w.run()
	return w, nil
}

func (d *Discovery) Close() error {
	return d.client.Close()
}

type watcher struct {
	d      *Discovery
	name   string
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan []*registry.ServiceInstance
	last   string
}

func (w *watcher) Next() ([]*registry.ServiceInstance, error) {
	instances, ok := <-w.ch
	if !ok {
		return nil, errors.New("registry: watch closed")
	}
	return instances, nil
}

func (w *watcher) Stop() error {
	w.cancel()
	return nil
}

func (w *watcher) run() {
	defer close(w.ch)

	// Watch every group in the resolution order; any change recomputes the
	// resolved snapshot via GetService.
	events := make(chan struct{}, 1)
	for _, g := range w.d.groups() {
		go w.watchPrefix(g, events)
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-events:
		case <-ticker.C:
		}
		instances, err := w.d.GetService(context.Background(), w.name)
		if err != nil {
			continue
		}
		hash := snapshotHash(instances)
		if hash == w.last {
			continue // no change, keep waiting
		}
		w.last = hash
		select {
		case w.ch <- instances:
		case <-w.ctx.Done():
			return
		}
	}
}

// watchPrefix forwards change events for one group's prefix to events.
func (w *watcher) watchPrefix(group string, events chan<- struct{}) {
	prefix := w.d.prefix(group, w.name)
	watchCh := w.d.client.Watch(w.ctx, prefix, clientv3.WithPrefix())
	for {
		select {
		case <-w.ctx.Done():
			return
		case resp, ok := <-watchCh:
			if !ok {
				return
			}
			if err := resp.Err(); err != nil {
				time.Sleep(time.Second)
				watchCh = w.d.client.Watch(w.ctx, prefix, clientv3.WithPrefix())
				continue
			}
			if len(resp.Events) == 0 {
				continue // watch keepalive, not a change
			}
			select {
			case events <- struct{}{}:
			case <-w.ctx.Done():
				return
			}
		}
	}
}

// snapshotHash normalizes ordering so identical instance sets compare equal.
func snapshotHash(instances []*registry.ServiceInstance) string {
	sort.Slice(instances, func(i, j int) bool { return instances[i].ID < instances[j].ID })
	b, _ := json.Marshal(instances)
	return string(b)
}
