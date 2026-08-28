// Package registry defines the service registration and discovery interfaces.
//
// Implementations and consumers must honor these requirements:
//
//   - Startup gate: a service must not start if the registry is unreachable.
//     Connectivity is verified before any listener binds, so an outage surfaces
//     at startup (Check) instead of as a late registration or resolve error.
//   - Local cache: a consumer keeps a local per-service cache refreshed by a
//     watch, so instances appear as soon as they register and disappear when
//     they leave, without a lookup on every request.
//   - Empty/error safety: a failed query must never clear the cache, and a
//     single empty observation must not immediately evict a known-good set.
//     An empty result is confirmed (re-queried after a delay) before the cache
//     entry is dropped, so a briefly-empty registry (e.g. restart before
//     instances re-register) keeps the consumer serving its cached backends.
package registry

import "context"

// DefaultGroup is the group regular instances register under. A caller that
// sets no preferred group (see WithGroup) resolves to DefaultGroup; a
// developer or canary instance registers under its own group to override it
// locally.
const DefaultGroup = "default"

// ServiceInstance is a single running instance of a service.
type ServiceInstance struct {
	ID            string            // unique instance ID, e.g. "ip:pid"
	Name          string            // service name, e.g. "user"
	VersionGit    string            // git/build version, injected at build time (-ldflags)
	VersionSchema string            // proto schema hash, from generated registry.pb.go
	Group         string            // registration group; "" registers under DefaultGroup
	Protocol      string            // caller protocol this instance advertises, e.g. "rpcx"
	Weight        int               // load-balancing weight; 0 means default
	Metadata      map[string]string
	Endpoints     []string          // "grpc://10.0.0.1:8502", "rpcx://10.0.0.1:8503"
}

// Registrar registers a service instance and keeps its lease alive until
// Deregister or Close. Heartbeat is owned by the implementation, not the App.
type Registrar interface {
	// Check verifies the registry is reachable; the App calls it at startup
	// (fail-fast gate) before any listener binds.
	Check(ctx context.Context) error
	Register(ctx context.Context, instance *ServiceInstance) error
	Deregister(ctx context.Context, instance *ServiceInstance) error
	Close() error
}

// Discovery finds and watches service instances.
type Discovery interface {
	// Check verifies the registry is reachable; consumers call it at startup
	// (fail-fast gate) so an outage surfaces before traffic is accepted.
	Check(ctx context.Context) error
	GetService(ctx context.Context, name string) ([]*ServiceInstance, error)
	Watch(ctx context.Context, name string) (Watcher, error)
	Close() error
}

// Watcher streams the current instance set; Next blocks until the set changes.
type Watcher interface {
	Next() ([]*ServiceInstance, error)
	Stop() error
}
