package etcd

import "time"

type Option func(*Registrar)

// WithTTL sets the lease duration; KeepAlive renews it until Deregister.
func WithTTL(d time.Duration) Option {
	return func(r *Registrar) { r.ttl = d }
}

// WithPrefix overrides the key prefix (used by tests to isolate).
func WithPrefix(p string) Option {
	return func(r *Registrar) { r.keyPrefix = p }
}

// DiscoveryOption configures a Discovery.
type DiscoveryOption func(*Discovery)

// WithGroup sets the preferred registration group. GetService and Watch
// resolve [group, DefaultGroup]: instances in group win, otherwise the
// default group (see Discovery).
func WithGroup(g string) DiscoveryOption {
	return func(d *Discovery) { d.group = g }
}
