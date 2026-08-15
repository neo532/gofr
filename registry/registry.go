// Package registry defines the service registration and discovery interfaces.
package registry

import "context"

// DefaultGroup is the group regular instances register under. A caller that
// sets no preferred group (see WithGroup) resolves to DefaultGroup; a
// developer or canary instance registers under its own group to override it
// locally.
const DefaultGroup = "default"

// ServiceInstance is a single running instance of a service.
type ServiceInstance struct {
	ID        string            // unique instance ID, e.g. "ip:pid"
	Name      string            // service name, e.g. "user"
	Version   string
	Group     string            // registration group; "" registers under DefaultGroup
	Protocol  string            // caller protocol this instance advertises, e.g. "rpcx"
	Weight    int               // load-balancing weight; 0 means default
	Metadata  map[string]string
	Endpoints []string          // "grpc://10.0.0.1:8502", "rpcx://10.0.0.1:8503"
}

// Registrar registers a service instance and keeps its lease alive until
// Deregister or Close. Heartbeat is owned by the implementation, not the App.
type Registrar interface {
	Register(ctx context.Context, instance *ServiceInstance) error
	Deregister(ctx context.Context, instance *ServiceInstance) error
	Close() error
}

// Discovery finds and watches service instances.
type Discovery interface {
	GetService(ctx context.Context, name string) ([]*ServiceInstance, error)
	Watch(ctx context.Context, name string) (Watcher, error)
	Close() error
}

// Watcher streams the current instance set; Next blocks until the set changes.
type Watcher interface {
	Next() ([]*ServiceInstance, error)
	Stop() error
}
