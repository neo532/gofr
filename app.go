package gofr

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/upgrader"
	"github.com/neo532/gokit/errorx"
	"github.com/neo532/gokit/logger"
)

// App manages server lifecycle.
type App struct {
	opts     *options
	cancel   context.CancelFunc
	instance *registry.ServiceInstance
}

// Logger returns the application logger.
func (a *App) Logger() logger.ILogger {
	return a.opts.logger
}

// New creates an App.
func New(opts ...Option) (a *App) {
	o := &options{
		ctx:              context.Background(),
		sigs:             []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT},
		stopTimeout:      10 * time.Second,
		registrarTimeout: 10 * time.Second,
		readyTimeout:     30 * time.Second,
		pidFile:          "./pid",
		logger:           logger.NewDefaultILogger(),
	}
	for _, opt := range opts {
		opt(o)
	}

	a = &App{opts: o}
	o.ctx, a.cancel = context.WithCancel(o.ctx)
	return
}

// Run starts all servers and blocks until a signal or a server error.
func (a *App) Run() error {
	eg, ctx := errgroup.WithContext(a.opts.ctx)

	// Create upgrader and inject listeners if enabled.
	var upg *upgrader.Upgrader
	if a.opts.enableUpgrader {
		upg = upgrader.New()
		for _, srv := range a.opts.servers {
			if ls, ok := srv.(transport.ListenerServer); ok {
				lis, err := upg.Listen("tcp", ls.Addr())
				if err != nil {
					return errorx.Wrap(err)
				}
				ls.SetListener(lis)
			}
		}
	}

	// inject the App into servers so they can reach shared resources (logger, etc.)
	for _, srv := range a.opts.servers {
		srv.App(a)
	}

	// beforeStart hooks
	for _, fn := range a.opts.beforeStart {
		if err := fn(ctx); err != nil {
			return errorx.Wrap(err)
		}
	}

	// Fail fast: the registry must be reachable before any listener binds, so an
	// outage surfaces at startup instead of after servers accept traffic.
	if a.opts.registrar != nil {
		checkCtx, cancel := context.WithTimeout(ctx, a.opts.registrarTimeout)
		if err := a.opts.registrar.Check(checkCtx); err != nil {
			cancel()
			return errorx.Wrap(err)
		}
		cancel()
	}

	// start servers
	for _, srv := range a.opts.servers {
		s := srv
		eg.Go(func() error {
			<-ctx.Done()
			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.opts.stopTimeout)
			defer cancel()
			return s.Stop(stopCtx)
		})
		eg.Go(func() error {
			return s.Start(ctx)
		})
	}

	// Wait until every server's listener is bound, then register. A server that
	// failed to start cancels ctx, so waitReady returns and eg.Wait surfaces the
	// real error; a server that never signals ready (listener stuck) times out.
	if err := a.waitReady(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			if werr := eg.Wait(); werr != nil && !errors.Is(werr, context.Canceled) {
				return werr
			}
			return nil
		}
		a.cancel()
		return err
	}

	if err := a.register(); err != nil {
		a.cancel()
		return errorx.Wrap(err)
	}

	// afterStart hooks
	for _, fn := range a.opts.afterStart {
		if err := fn(ctx); err != nil {
			return errorx.Wrap(err)
		}
	}

	// Write PID file after all init (including afterStart health checks) succeed.
	if err := a.WritePID(); err != nil {
		return errorx.Wrap(err)
	}

	// If this is the upgraded child, signal parent we're ready.
	// This runs after servers have started (their Start goroutines are
	// running and listeners are accepting).
	if upg != nil && upg.IsChild() {
		if err := upg.Ready(); err != nil {
			return errorx.Wrap(err)
		}
	}

	// signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, a.opts.sigs...)
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case <-quit:
			return a.Stop()
		}
	})

	// SIGHUP — graceful restart via fd inheritance
	if upg != nil && !upg.IsChild() {
		eg.Go(func() error {
			hup := make(chan os.Signal, 1)
			signal.Notify(hup, syscall.SIGHUP)
			select {
			case <-hup:
				if err := upg.Upgrade(); err != nil {
					return err
				}
				return a.Stop()
			case <-ctx.Done():
				return nil
			}
		})
	}

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	for _, fn := range a.opts.afterStop {
		if err := fn(ctx); err != nil {
			return errorx.Wrap(err)
		}
	}
	return errorx.Wrap(a.unregister())
}

// Stop gracefully stops the application.
func (a *App) Stop() error {
	for _, fn := range a.opts.beforeStop {
		if err := fn(a.opts.ctx); err != nil {
			return err
		}
	}
	a.cancel()
	return nil
}

// waitReady blocks until every ReadyServer has bound its listener, so a
// registration never outlives a server that failed to start. When a server
// errors, the errgroup cancels ctx and this returns ctx.Err(); when a server
// never signals ready within readyTimeout, it returns a timeout error.
func (a *App) waitReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.opts.readyTimeout)
	defer cancel()
	for _, srv := range a.opts.servers {
		rs, ok := srv.(transport.ReadyServer)
		if !ok {
			continue
		}
		select {
		case <-rs.Ready():
		case <-ctx.Done():
			return errorx.Wrapf(ctx.Err(), "wait ready for %T", srv)
		}
	}
	return nil
}

// register builds the service instance from the servers' advertised endpoints
// and registers it. The registrar owns the lease-renewal heartbeat, so a crash
// clears the entry via lease expiry. A failed registration aborts startup.
func (a *App) register() error {
	if a.opts.registrar == nil {
		return nil
	}
	inst, err := a.buildInstance()
	if err != nil {
		return errorx.Wrap(err)
	}
	ctx, cancel := context.WithTimeout(a.opts.ctx, a.opts.registrarTimeout)
	defer cancel()
	if err := a.opts.registrar.Register(ctx, inst); err != nil {
		return errorx.Wrapf(err, "register %s", inst.Name)
	}
	a.instance = inst
	return nil
}

// unregister deregisters the instance and closes the registrar, releasing the
// connection it owns. It runs during shutdown, when a.opts.ctx has already been
// cancelled by Stop, so it derives its own uncancelled context.
func (a *App) unregister() error {
	if a.opts.registrar == nil {
		return nil
	}
	var err error
	if a.instance != nil {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(a.opts.ctx), a.opts.registrarTimeout)
		defer cancel()
		err = a.opts.registrar.Deregister(ctx, a.instance)
	}
	if cerr := a.opts.registrar.Close(); cerr != nil && err == nil {
		err = cerr
	}
	return errorx.Wrap(err)
}

// buildInstance assembles the ServiceInstance from each server's advertised
// endpoint. The instance ID falls back to hostname:pid when not configured.
func (a *App) buildInstance() (*registry.ServiceInstance, error) {
	group := a.opts.group
	if group == "" {
		group = registry.DefaultGroup
	}
	inst := &registry.ServiceInstance{
		Name:          a.opts.name,
		VersionGit:    a.opts.versionGit,
		VersionSchema: a.opts.schemaVersion,
		Group:         group,
		Protocol:      a.opts.protocol,
		Weight:        a.opts.weight,
		Metadata:      a.opts.metadata,
		Endpoints:     []string{},
	}
	if a.opts.id != "" {
		inst.ID = a.opts.id
	} else {
		host, err := os.Hostname()
		if err != nil {
			return nil, errorx.Wrap(err)
		}
		inst.ID = host + ":" + strconv.Itoa(os.Getpid())
	}
	for _, srv := range a.opts.servers {
		ep, ok := srv.(transport.Endpointer)
		if !ok {
			continue
		}
		u, err := ep.Endpoint()
		if err != nil {
			return nil, errorx.Wrapf(err, "endpoint %T", srv)
		}
		inst.Endpoints = append(inst.Endpoints, u.String())
	}
	return inst, nil
}

func (a *App) WritePID() (err error) {
	p := strconv.Itoa(os.Getpid())

	var f *os.File
	f, err = os.OpenFile(a.opts.pidFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	if err != nil {
		err = errorx.Wrap(err)
		return
	}
	var n int
	n, err = f.Write([]byte(p))
	if err != nil {
		err = errorx.Wrap(err)
		return
	}
	if n < len(p) {
		err = errorx.Wrap(io.ErrShortWrite)
	}
	return
}
