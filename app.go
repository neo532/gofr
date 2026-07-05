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

	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/upgrader"
)

// App manages server lifecycle.
type App struct {
	opts   *options
	cancel context.CancelFunc
}

// New creates an App.
func New(opts ...Option) (a *App) {
	o := &options{
		ctx:         context.Background(),
		sigs:        []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT},
		stopTimeout: 10 * time.Second,
		pidFile:     "./pid",
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
					return err
				}
				ls.SetListener(lis)
			}
		}
	}

	// beforeStart hooks
	for _, fn := range a.opts.beforeStart {
		if err := fn(ctx); err != nil {
			return err
		}
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

	// afterStart hooks
	for _, fn := range a.opts.afterStart {
		if err := fn(ctx); err != nil {
			return err
		}
	}

	// Write PID file after all init (including afterStart health checks) succeed.
	if err := a.WritePID(); err != nil {
		return err
	}

	// If this is the upgraded child, signal parent we're ready.
	// This runs after servers have started (their Start goroutines are
	// running and listeners are accepting).
	if upg != nil && upg.IsChild() {
		if err := upg.Ready(); err != nil {
			return err
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
			return err
		}
	}
	return nil
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
		return
	}
	var n int
	n, err = f.Write([]byte(p))
	if err != nil {
		return
	}
	if n < len(p) {
		err = io.ErrShortWrite
	}
	return
}
