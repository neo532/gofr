package gofr

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/neo532/gofr/logger"
	"github.com/neo532/gofr/transport"
)

type Option func(o *options)

type options struct {
	id        string
	name      string
	version   string
	metadata  map[string]string
	endpoints []string

	ctx  context.Context
	sigs []os.Signal

	logger      logger.Logger
	stopTimeout time.Duration
	servers     []transport.Server

	beforeStart []func(context.Context) error
	beforeStop  []func(context.Context) error
	afterStart  []func(context.Context) error
	afterStop   []func(context.Context) error
}

// ID with service id.
func ID(id string) Option {
	return func(o *options) { o.id = id }
}

// Name with service name.
func Name(name string) Option {
	return func(o *options) { o.name = name }
}

// Version with service version.
func Version(version string) Option {
	return func(o *options) { o.version = version }
}

// Metadata with service metadata.
func Metadata(md map[string]string) Option {
	return func(o *options) { o.metadata = md }
}

// Endpoint with service endpoint.
func Endpoint(endpoints ...*url.URL) Option {
	return func(o *options) {
		o.endpoints = make([]string, 0, len(endpoints))
		for _, e := range endpoints {
			o.endpoints = append(o.endpoints, e.String())
		}
	}
}

// Context with service context.
func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

// Logger with service logger.
func Logger(logger logger.Logger) Option {
	return func(o *options) { o.logger = logger }
}

// Server with transport servers.
func Server(srv ...transport.Server) Option {
	return func(o *options) { o.servers = srv }
}

// Signal with exit signals.
func Signal(sigs ...os.Signal) Option {
	return func(o *options) { o.sigs = sigs }
}

// StopTimeout with app stop timeout.
func StopTimeout(t time.Duration) Option {
	return func(o *options) { o.stopTimeout = t }
}

// BeforeStart run funcs before app starts
func BeforeStart(fns ...func(context.Context) error) Option {
	return func(o *options) {
		o.beforeStart = fns
	}
}

// BeforeStop run funcs before app stops
func BeforeStop(fns ...func(context.Context) error) Option {
	return func(o *options) {
		o.beforeStop = fns
	}
}

// AfterStart run funcs after app starts
func AfterStart(fns ...func(context.Context) error) Option {
	return func(o *options) {
		o.afterStart = fns
	}
}

// AfterStop run funcs after app stops
func AfterStop(fns ...func(context.Context) error) Option {
	return func(o *options) {
		o.afterStop = fns
	}
}
