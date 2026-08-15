package redis

import "time"

type config struct {
	keyPrefix  string
	chanPrefix string
	ttl        time.Duration
	group      string
}

type Option func(*config)

// WithTTL sets the lease duration; the heartbeat renews it every TTL/3.
func WithTTL(d time.Duration) Option {
	return func(c *config) { c.ttl = d }
}

// WithGroup sets the preferred registration group. GetService and Watch
// resolve [group, DefaultGroup]: instances in group win, otherwise the
// default group.
func WithGroup(g string) Option {
	return func(c *config) { c.group = g }
}

// WithPrefix overrides the key and channel prefixes (used by tests to isolate).
func WithPrefix(key, channel string) Option {
	return func(c *config) {
		if key != "" {
			c.keyPrefix = key
		}
		if channel != "" {
			c.chanPrefix = channel
		}
	}
}

func defaultConfig() config {
	return config{keyPrefix: keyPrefix, chanPrefix: chanPrefix, ttl: defaultTTL}
}
