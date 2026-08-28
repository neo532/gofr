package transport

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

// Operation is the observable identity of a request, uniform across protocols.
// It is a read-only value: each Transporter returns a fresh copy per call.
type Operation struct {
	// Operation is the proto route template, e.g. "/cms/appconfig/:namespace/:key/content".
	// For gRPC/rpcx methods without a google.api.http binding it falls back to the
	// full method name ("/pkg.Service/Method").
	Operation string
	// Path is the real path with actual parameter values, e.g. "/cms/appconfig/11/1/content".
	// HTTP/WebSocket read it from the request URL; gRPC/rpcx substitute values from
	// the decoded request into the route template. Empty when no HTTP binding exists.
	Path string
	// Method is the proto-defined HTTP verb (GET/POST/PUT/DELETE/PATCH), uniform
	// across protocols. Empty for gRPC/rpcx methods without a binding.
	Method string
}

// Transporter is request-scoped transport context.
type Transporter interface {
	Kind() Kind
	Endpoint() string
	Operation() Operation
	RequestHeader() Header
	ReplyHeader() Header
	App() App
	// ClientIP returns the real client IP, resolving proxy headers per the
	// server's TrustedProxies configuration (see ip.ClientIP).
	ClientIP() string
}
