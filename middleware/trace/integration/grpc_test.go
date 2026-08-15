package integration

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	gofrTrace "github.com/neo532/gofr/middleware/trace"
	gofrgrpc "github.com/neo532/gofr/transport/grpc"
)

type grpcReq struct {
	Name string
}

type grpcReply struct {
	Message string
}

type grpcSvc struct{}

func TestGRPCTracePropagation(t *testing.T) {
	srv := gofrgrpc.NewServer(gofrgrpc.Address(":0"), gofrgrpc.Middleware(gofrTrace.Server()))
	gofrgrpc.RegisterServiceWith(srv, "test.Trace", &grpcSvc{}, []struct {
		Name    string
		NewReq  func() any
		Handler gofrgrpc.UnaryHandler
	}{
		{
			Name:   "TraceID",
			NewReq: func() any { return &grpcReq{} },
			Handler: func(ctx context.Context, req any) (any, error) {
				return &grpcReply{Message: gofrTrace.TraceID(ctx)}, nil
			},
		},
	})

	addr, stop := start(t, srv)
	defer stop()

	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reply := &grpcReply{}

	// Incoming traceparent metadata → handler sees the same trace ID.
	ctx := metadata.AppendToOutgoingContext(context.Background(), "traceparent", traceparentHeader)
	if err := conn.Invoke(ctx, "/test.Trace/TraceID", &grpcReq{}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != wantTraceID {
		t.Fatalf("incoming traceparent: got %q, want %q", reply.Message, wantTraceID)
	}

	// No metadata → generated valid 32-hex ID.
	if err := conn.Invoke(context.Background(), "/test.Trace/TraceID", &grpcReq{}, reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.Message) != 32 || reply.Message == "00000000000000000000000000000000" {
		t.Fatalf("generated trace ID: got %q, want 32-hex non-zero", reply.Message)
	}
}
