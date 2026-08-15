package integration

import (
	"context"
	"testing"

	rpcxClient "github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/share"

	gofrTrace "github.com/neo532/gofr/middleware/trace"
	rpcx "github.com/neo532/gofr/transport/rpcx"
)

// RPCXArgs/RPCXReply must be exported for rpcx reflection.
type RPCXArgs struct {
	Name string
}

type RPCXReply struct {
	Message string
}

// TraceService reports the active trace ID back through the reply.
type TraceService struct{}

func (s *TraceService) TraceID(ctx context.Context, args *RPCXArgs, reply *RPCXReply) error {
	reply.Message = gofrTrace.TraceID(ctx)
	return nil
}

func TestRPCXTracePropagation(t *testing.T) {
	srv := rpcx.NewServer(rpcx.Address(":0"), rpcx.Middleware(gofrTrace.Server()))
	rpcx.RegisterServiceWith(srv, "TraceService", &TraceService{})

	addr, stop := start(t, srv)
	defer stop()

	d, err := rpcxClient.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		t.Fatal(err)
	}
	c := rpcxClient.NewOneClient(rpcxClient.Failtry, rpcxClient.RandomSelect, d, rpcxClient.DefaultOption)
	defer c.Close()

	reply := &RPCXReply{}

	// Incoming traceparent metadata → handler sees the same trace ID.
	ctx := context.WithValue(context.Background(), share.ReqMetaDataKey, map[string]string{"traceparent": traceparentHeader})
	if err := c.Call(ctx, "TraceService", "TraceID", &RPCXArgs{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != wantTraceID {
		t.Fatalf("incoming traceparent: got %q, want %q", reply.Message, wantTraceID)
	}

	// No metadata → generated valid 32-hex ID.
	if err := c.Call(context.Background(), "TraceService", "TraceID", &RPCXArgs{}, reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.Message) != 32 || reply.Message == "00000000000000000000000000000000" {
		t.Fatalf("generated trace ID: got %q, want 32-hex non-zero", reply.Message)
	}
}
