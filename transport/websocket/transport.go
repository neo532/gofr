package websocket

import (
	"net/http"

	"github.com/neo532/gofr/transport"
)

// wsTransport implements transport.Transporter for WebSocket connections.
type wsTransport struct {
	endpoint  string
	operation string
	reqHeader headerCarrier
}

func (t *wsTransport) Kind() transport.Kind          { return transport.KindWebSocket }
func (t *wsTransport) Endpoint() string               { return t.endpoint }
func (t *wsTransport) Operation() string               { return t.operation }
func (t *wsTransport) RequestHeader() transport.Header  { return t.reqHeader }
func (t *wsTransport) ReplyHeader() transport.Header    { return nil }

// headerCarrier adapts http.Header to transport.Header.
type headerCarrier http.Header

func (h headerCarrier) Get(key string) string      { return http.Header(h).Get(key) }
func (h headerCarrier) Set(key, value string)       { http.Header(h).Set(key, value) }
func (h headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range http.Header(h) {
		keys = append(keys, k)
	}
	return keys
}
