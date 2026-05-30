package transport

import (
	"context"
	"net/url"
)

// Server is transport server lifecycle.
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

// Endpointer returns registry endpoint.
type Endpointer interface {
	Endpoint() (*url.URL, error)
}

// Kind defines transport type.
type Kind string

const (
	KindHTTP Kind = "http"
	KindGRPC Kind = "grpc"
	KindWebSocket Kind = "ws"
)

// Header is the storage medium used by a Transporter.
type Header interface {
	Get(key string) string
	Set(key string, value string)
	Keys() []string
}

// Transporter is request-scoped transport context.
type Transporter interface {
	Kind() Kind
	Endpoint() string
	Operation() string
	RequestHeader() Header
	ReplyHeader() Header
}

type serverTransportKey struct{}

func NewServerContext(ctx context.Context, tr Transporter) context.Context {
	return context.WithValue(ctx, serverTransportKey{}, tr)
}

func FromServerContext(ctx context.Context) (Transporter, bool) {
	tr, ok := ctx.Value(serverTransportKey{}).(Transporter)
	return tr, ok
}
