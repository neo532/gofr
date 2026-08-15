package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/neo532/gofr/registry"
)

type Discovery struct {
	client *redis.Client
	cfg    config
}

func NewDiscovery(addr string, opts ...Option) *Discovery {
	d := &Discovery{
		client: redis.NewClient(&redis.Options{Addr: addr, PoolSize: 20}),
		cfg:    defaultConfig(),
	}
	for _, o := range opts {
		o(&d.cfg)
	}
	return d
}

// groups returns the group resolution order: [preferred, default].
func (d *Discovery) groups() []string {
	if d.cfg.group == "" || d.cfg.group == registry.DefaultGroup {
		return []string{registry.DefaultGroup}
	}
	return []string{d.cfg.group, registry.DefaultGroup}
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
	pattern := d.cfg.keyPrefix + group + ":" + name + ":*"
	var instances []*registry.ServiceInstance
	iter := d.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		data, err := d.client.Get(ctx, iter.Val()).Bytes()
		if err == redis.Nil {
			continue // lease expired between scan and get
		}
		if err != nil {
			return nil, err
		}
		var inst registry.ServiceInstance
		if err := json.Unmarshal(data, &inst); err != nil {
			return nil, err
		}
		instances = append(instances, &inst)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return instances, nil
}

func (d *Discovery) Watch(ctx context.Context, name string) (registry.Watcher, error) {
	channels := make([]string, 0, 2)
	for _, g := range d.groups() {
		channels = append(channels, d.cfg.chanPrefix+g+":"+name)
	}
	pubsub := d.client.Subscribe(ctx, channels...)
	if _, err := pubsub.Receive(ctx); err != nil {
		pubsub.Close()
		return nil, err
	}
	wCtx, cancel := context.WithCancel(ctx)
	w := &watcher{
		d:      d,
		name:   name,
		ctx:    wCtx,
		cancel: cancel,
		pubsub: pubsub,
		ticker: time.NewTicker(d.cfg.ttl / 3),
		msgCh:  make(chan struct{}, 1),
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
	pubsub *redis.PubSub
	ticker *time.Ticker
	msgCh  chan struct{}
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
	w.ticker.Stop()
	return w.pubsub.Close()
}

func (w *watcher) run() {
	defer close(w.ch)

	// Best-effort pub/sub receiver feeding msgCh. Polling remains the source
	// of truth so a missed or dropped pub/sub message cannot stall discovery.
	go func() {
		for {
			_, err := w.pubsub.ReceiveMessage(w.ctx)
			if err != nil {
				if w.ctx.Err() != nil {
					return
				}
				time.Sleep(time.Second)
				continue
			}
			select {
			case w.msgCh <- struct{}{}:
			case <-w.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.ticker.C:
		case <-w.msgCh:
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

// snapshotHash normalizes ordering so identical instance sets compare equal.
func snapshotHash(instances []*registry.ServiceInstance) string {
	sort.Slice(instances, func(i, j int) bool { return instances[i].ID < instances[j].ID })
	b, _ := json.Marshal(instances)
	return string(b)
}
