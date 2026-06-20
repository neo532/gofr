package gofr

import (
	"context"
	"os"
	"time"

	"github.com/neo532/gofr/transport"
	"github.com/neo532/gokit/logger"
)

// Option configures the App.
type Option func(o *options)

type options struct {
	id       string
	name     string
	version  string
	metadata map[string]string

	ctx  context.Context
	sigs []os.Signal

	stopTimeout time.Duration
	logger      logger.ILogger
	servers     []transport.Server

	// Before and After funcs
	beforeStart []func(context.Context) error
	afterStart  []func(context.Context) error
	beforeStop  []func(context.Context) error
	afterStop   []func(context.Context) error

	// endpoints []*url.URL
	// registrar        registry.Registrar
	// registrarTimeout time.Duration
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

func Metadata(md map[string]string) Option {
	return func(o *options) { o.metadata = md }
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

func Logger(l logger.ILogger) Option {
	return func(o *options) { o.logger = l }
}

func Server(srv ...transport.Server) Option {
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
