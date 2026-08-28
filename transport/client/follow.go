package client

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gokit/logger"
)

// DialFunc builds a protocol handle from the current instance set. It is the
// per-protocol dial step, closed over the service's rpcx service names and the
// app's middleware/retry options. cleanup releases the handle when a newer
// instance set replaces it.
type DialFunc func(ctx context.Context, instances []*registry.ServiceInstance) (handle any, baseURL string, cleanup func(), err error)

// Dialers collects the per-protocol DialFuncs for one followed service. The app
// builds it once, closing each protocol's dial helper over its options and, for
// rpcx, the service's rpcx service names. The protocol is then fixed for the
// lifetime of the FollowClient. Logger is used for background-goroutine panics.
type Dialers struct {
	GRPC   DialFunc
	RPCX   DialFunc
	HTTP   DialFunc
	WS     DialFunc
	Logger logger.ILogger
}

// Dial returns the DialFunc for kind, or an error when the protocol is not
// wired in d.
func (d Dialers) Dial(kind transport.Kind) (DialFunc, error) {
	switch kind {
	case transport.KindGRPC:
		return d.GRPC, nil
	case transport.KindRPCX:
		return d.RPCX, nil
	case transport.KindHTTP:
		return d.HTTP, nil
	case transport.KindWebSocket:
		return d.WS, nil
	}
	return nil, fmt.Errorf("transport: no dialer for protocol %q", kind)
}

// clientState bundles a dialed client with the cleanup that releases its
// protocol handle, so FollowClient can swap and close atomically.
type clientState[T any] struct {
	uc      T
	cleanup func()
}

// FollowClient keeps a typed client wired to the latest instance set the
// registry reports for the followed service. It dials once at startup and
// re-dials only when Watch signals an instance-set change, so no request ever
// hits the registry. The protocol is fixed at startup (a change takes effect on
// restart); only addresses and weights follow live.
type FollowClient[T any] struct {
	svc      string
	kind     transport.Kind
	dial     DialFunc
	assemble func(kind string, handle any, baseURL string) T
	logger   logger.ILogger
	state    atomic.Pointer[clientState[T]]
	stop     chan struct{}
	done     chan struct{}
}

// NewFollowClient wires a FollowClient to svc through the registry. The
// protocol is read once from the first advertised instance and fixed for the
// lifetime of the client; only the instance set is followed afterwards. The
// returned cleanup closes the FollowClient and its protocol handle.
func NewFollowClient[T any](ctx context.Context, disc registry.Discovery, svc string, dials Dialers, assemble func(kind string, handle any, baseURL string) T) (*FollowClient[T], func(), error) {
	instances, err := disc.GetService(ctx, svc)
	if err != nil {
		return nil, nil, err
	}
	if len(instances) == 0 {
		return nil, nil, fmt.Errorf("registry: no instance for %q", svc)
	}
	kind := transport.Kind(instances[0].Protocol)
	dial, err := dials.Dial(kind)
	if err != nil {
		return nil, nil, err
	}
	log := dials.Logger
	if log == nil {
		log = logger.NewDefaultILogger()
	}
	fc := &FollowClient[T]{
		svc:      svc,
		kind:     kind,
		dial:     dial,
		assemble: assemble,
		logger:   log,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	if err := fc.redial(ctx, instances); err != nil {
		return nil, nil, err
	}
	go fc.run(ctx, disc)
	return fc, fc.Close, nil
}

// Client returns the current typed client, atomically swapped on re-dial. It is
// safe to call concurrently; the returned client stays valid even if a re-dial
// happens in the background.
func (f *FollowClient[T]) Client() T {
	if s := f.state.Load(); s != nil {
		return s.uc
	}
	var zero T
	return zero
}

// Close stops the watch and releases the current protocol handle.
func (f *FollowClient[T]) Close() {
	select {
	case <-f.stop:
		return
	default:
		close(f.stop)
		<-f.done
	}
	if s := f.state.Load(); s != nil && s.cleanup != nil {
		s.cleanup()
	}
}

// recoverPanic logs a goroutine panic with its stack instead of crashing the
// process. Defer it first thing in every goroutine this type spawns.
func (f *FollowClient[T]) recoverPanic(tag string) {
	if r := recover(); r != nil {
		f.logger.Error(context.Background(), "gofr: "+tag+" panic", "panic", r, "stack", string(debug.Stack()))
	}
}

// redial builds a fresh client from instances and swaps it in, closing the
// previous handle. The dial func is already fixed to the startup kind, so a
// protocol change is not followed at runtime — it takes effect on restart.
func (f *FollowClient[T]) redial(ctx context.Context, instances []*registry.ServiceInstance) error {
	handle, baseURL, cleanup, err := f.dial(ctx, instances)
	if err != nil {
		return err
	}
	old := f.state.Swap(&clientState[T]{uc: f.assemble(string(f.kind), handle, baseURL), cleanup: cleanup})
	if old != nil && old.cleanup != nil {
		old.cleanup()
	}
	return nil
}

// run drives the registry watch: every instance-set change re-dials and swaps
// in a new client. It returns when the watcher closes or Stop is called.
func (f *FollowClient[T]) run(ctx context.Context, disc registry.Discovery) {
	defer close(f.done)
	defer f.recoverPanic("FollowClient.run")
	wt, err := disc.Watch(ctx, f.svc)
	if err != nil {
		return
	}
	defer wt.Stop()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		defer f.recoverPanic("FollowClient.watch")
		select {
		case <-f.stop:
			_ = wt.Stop()
		case <-stop:
		}
	}()
	for {
		instances, err := wt.Next()
		if err != nil {
			return
		}
		_ = f.redial(ctx, instances)
	}
}
