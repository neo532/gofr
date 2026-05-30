package core

import (
	"context"
	"os"
	"time"

	"github.com/neo532/gofr/transport"
)

// Option configures the App.
type Option func(o *options)

type options struct {
	id      string
	name    string
	version string

	ctx  context.Context
	sigs []os.Signal

	stopTimeout time.Duration
	servers     []transport.Server

	beforeStart []func(context.Context) error
	afterStart  []func(context.Context) error
	beforeStop  []func(context.Context) error
	afterStop   []func(context.Context) error
}

func ID(id string) Option {
	return func(o *options) { o.id = id }
}

func Name(name string) Option {
	return func(o *options) { o.name = name }
}

func Version(v string) Option {
	return func(o *options) { o.version = v }
}

func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

func Signal(sigs ...os.Signal) Option {
	return func(o *options) { o.sigs = sigs }
}

func StopTimeout(d time.Duration) Option {
	return func(o *options) { o.stopTimeout = d }
}

func WithServer(srv ...transport.Server) Option {
	return func(o *options) { o.servers = append(o.servers, srv...) }
}

func BeforeStart(fn func(context.Context) error) Option {
	return func(o *options) { o.beforeStart = append(o.beforeStart, fn) }
}

func AfterStart(fn func(context.Context) error) Option {
	return func(o *options) { o.afterStart = append(o.afterStart, fn) }
}

func BeforeStop(fn func(context.Context) error) Option {
	return func(o *options) { o.beforeStop = append(o.beforeStop, fn) }
}

func AfterStop(fn func(context.Context) error) Option {
	return func(o *options) { o.afterStop = append(o.afterStop, fn) }
}

func applyDefaults(o *options) {
	if o.ctx == nil {
		o.ctx = context.Background()
	}
	if len(o.sigs) == 0 {
		o.sigs = []os.Signal{os.Interrupt}
	}
	if o.stopTimeout <= 0 {
		o.stopTimeout = 10 * time.Second
	}
}
