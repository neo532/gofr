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
)

// App manages server lifecycle.
type App struct {
	opts   *options
	cancel context.CancelFunc
}

// New creates an App.
func New(opts ...Option) *App {
	o := &options{
		ctx:         context.Background(),
		sigs:        []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT},
		stopTimeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(o)
	}

	_, cancel := context.WithCancel(o.ctx)
	return &App{opts: o, cancel: cancel}
}

// Run starts all servers and blocks until a signal or a server error.
func (a *App) Run() error {
	eg, ctx := errgroup.WithContext(a.opts.ctx)

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

func (a *App) WritePID(file string) (err error) {
	p := strconv.Itoa(os.Getpid())
	if file == "" {
		file = "./pid"
	}

	var f *os.File
	f, err = os.OpenFile(file, os.O_WRONLY|os.O_CREATE, os.ModePerm)
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
