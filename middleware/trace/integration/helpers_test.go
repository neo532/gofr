package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/grpc/encoding"

	"github.com/neo532/gofr/transport"
)

const traceparentHeader = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
const wantTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

// grpcJSONCodec lets the gRPC integration test use plain JSON messages instead
// of protobuf. Must be registered before any call; init() runs at load time.
type grpcJSONCodec struct{}

func (grpcJSONCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (grpcJSONCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (grpcJSONCodec) Name() string                       { return "json" }

func init() { encoding.RegisterCodec(grpcJSONCodec{}) }

// start runs a transport.Server on an ephemeral port and returns the bound
// address plus a stop func. Every gofr server exposes Addr() and blocks in
// Start(ctx) until the context is cancelled (rpcx returns immediately from
// Start, so its shutdown is driven by Stop below).
func start(t *testing.T, srv transport.Server) (addr string, stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	ls, ok := srv.(interface{ Addr() string })
	if !ok {
		t.Fatal("server does not expose Addr()")
	}
	for i := 0; i < 200; i++ {
		if a := ls.Addr(); a != "" && a != ":0" {
			addr = a
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		t.Fatal("server did not bind an address")
	}

	stop = func() {
		cancel()
		<-done
		stopCtx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer scancel()
		_ = srv.Stop(stopCtx)
	}
	return addr, stop
}
