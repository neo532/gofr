package gofr

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/neo532/gofr/logger"
	"github.com/neo532/gofr/logger/log"
)

// IApp is application context value.
type IApp interface {
	ID() string
	Name() string
	Version() string
	Metadata() map[string]string
	Endpoint() []string
}

type App struct {
	opts   *options
	cancel func()
}

func New(opts ...Option) (a *App) {

	a = &App{
		opts: &options{
			sigs:        []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT},
			stopTimeout: 10 * time.Second,
			logger:      logger.NewDefaultLogger(log.New()),
			ctx:         context.Background(),
		},
	}
	for _, opt := range opts {
		opt(a.opts)
	}
	a.opts.ctx, a.cancel = context.WithCancel(a.opts.ctx)

	if a.opts.id == "" {
		if id, err := uuid.NewUUID(); err == nil {
			a.opts.id = id.String()
		}
	}

	return
}

// ID returns app instance id.
func (a *App) ID() string { return a.opts.id }

// Name returns service name.
func (a *App) Name() string { return a.opts.name }

// Version returns app version.
func (a *App) Version() string { return a.opts.version }

// Metadata returns service metadata.
func (a *App) Metadata() map[string]string { return a.opts.metadata }

// Endpoint returns endpoints.
func (a *App) Endpoint() []string { return a.opts.endpoints }

func (a *App) Run() (err error) {

	for _, fn := range a.opts.beforeStart {
		if err = fn(a.opts.ctx); err != nil {
			return
		}
	}

	eg, _ := errgroup.WithContext(a.opts.ctx)

	wg := sync.WaitGroup{}
	for _, srv := range a.opts.servers {
		eg.Go(func() error {
			return srv.Start(a.opts.ctx)
		})

	}
	wg.Wait()

	for _, fn := range a.opts.afterStart {
		if err = fn(a.opts.ctx); err != nil {
			return
		}
	}

	return
}

func (a *App) Stop() (err error) {

	for _, fn := range a.opts.beforeStop {
		if er := fn(a.opts.ctx); er != nil {
			err = WrapErr(err, er)
		}
	}

	if a.cancel != nil {
		a.cancel()
	}

	for _, fn := range a.opts.afterStop {
		if er := fn(a.opts.ctx); er != nil {
			err = WrapErr(err, er)
		}
	}

	return
}

type iAppKey struct{}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, a IApp) context.Context {
	return context.WithValue(ctx, iAppKey{}, a)
}

// FromContext returns the Transport value stored in ctx, if any.
func FromContext(ctx context.Context) (a IApp, ok bool) {
	a, ok = ctx.Value(iAppKey{}).(IApp)
	return
}

func WrapErr(err error, er error) error {
	if err != nil {
		return errors.Wrap(err, er.Error())
	}
	return er
}
