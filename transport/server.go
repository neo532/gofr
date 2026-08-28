package transport

import (
	"context"
	"net"
	"net/url"

	"github.com/neo532/gokit/logger"
)

// App exposes shared application-level resources to transports. *gofr.App
// satisfies this interface; add methods here for anything a Transporter or a
// middleware needs from the application.
type App interface {
	Logger() logger.ILogger
}

// Server is transport server lifecycle.
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
	// App injects the application's shared resources (logger, etc.) into the
	// server so each request's Transporter can reach them via Transporter.App().
	App(App)
}

// ListenerServer is a Server that supports external listener injection
// for fd-inheritance graceful restart.
type ListenerServer interface {
	Server
	Addr() string
	SetListener(net.Listener)
}

// Endpointer returns registry endpoint.
type Endpointer interface {
	Endpoint() (*url.URL, error)
}

// ReadyServer is a Server that signals readiness once its listener is bound.
// The App waits on Ready() before registering, so a registration never
// outlives a server that failed to bind.
type ReadyServer interface {
	Server
	Ready() <-chan struct{}
}

// ServerTransportKey is the context key for storing/retrieving Transporter.
type ServerTransportKey struct{}

func NewServerContext(ctx context.Context, tr Transporter) context.Context {
	return context.WithValue(ctx, ServerTransportKey{}, tr)
}

func FromServerContext(ctx context.Context) (Transporter, bool) {
	tr, ok := ctx.Value(ServerTransportKey{}).(Transporter)
	return tr, ok
}
