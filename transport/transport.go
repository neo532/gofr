package transport

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/neo532/gokit/logger"
)

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

// EndpointURL resolves the advertised endpoint for a listen address. scheme is
// the transport kind ("http", "grpc", "rpcx", "ws"). When both the address
// host and the override are wildcards, the local IP is used.
func EndpointURL(scheme, address, override string) (*url.URL, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("endpoint %s: %q: %w", scheme, address, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = override
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = LocalIP()
	}
	if host == "" {
		return nil, fmt.Errorf("endpoint %s: cannot determine host for %q", scheme, address)
	}
	return &url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port)}, nil
}

// LogListen logs a server's bound listener address, protocol and local IP right
// after the listener is up, so an operator can see the service started
// normally. It is a no-op when the server is used standalone (no App injected),
// e.g. in tests.
func LogListen(ctx context.Context, app App, lis net.Listener, kind Kind) {
	if app == nil {
		return
	}
	app.Logger().Info(ctx, "listening on",
		KindKey, kind,
		"addr", lis.Addr().String(),
	)
}

// LocalIP returns the first non-loopback IPv4 address, the default advertised
// host for registration when no explicit host is configured.
func LocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		return ipnet.IP.String()
	}
	return ""
}

// Kind defines transport type.
type Kind string

const (
	KindKey = "kind"

	KindHTTP      Kind = "http"
	KindGRPC      Kind = "grpc"
	KindRPCX      Kind = "rpcx"
	KindWebSocket Kind = "ws"
	KindScript    Kind = "script"
)

// Header is the storage medium used by a Transporter.
type Header interface {
	Get(key string) string
	Set(key string, value string)
	Keys() []string
}

// App exposes shared application-level resources to transports. *gofr.App
// satisfies this interface; add methods here for anything a Transporter or a
// middleware needs from the application.
type App interface {
	Logger() logger.ILogger
}

// Transporter is request-scoped transport context.
type Transporter interface {
	Kind() Kind
	Endpoint() string
	Operation() string
	RequestHeader() Header
	ReplyHeader() Header
	App() App
	// ClientIP returns the real client IP, resolving proxy headers per the
	// server's TrustedProxies configuration (see transport.ClientIP).
	ClientIP() string
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
