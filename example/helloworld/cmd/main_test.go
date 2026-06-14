package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/neo532/gofr"
	pb "github.com/neo532/gofr/example/helloworld/api"
	"github.com/neo532/gofr/example/helloworld/service"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"

	gofrgrpc "github.com/neo532/gofr/transport/grpc"
	gofrhttp "github.com/neo532/gofr/transport/http"
	"github.com/neo532/gofr/transport/rpcx"

	rpcxClient "github.com/smallnest/rpcx/client"
)

// JSON codec for gRPC test (non-protobuf messages handled via json).
type testCodec struct{}

func (testCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (testCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (testCodec) Name() string                       { return "json" }

func init() { encoding.RegisterCodec(testCodec{}) }

func startApp(t *testing.T) (httpAddr, grpcAddr, rpcxAddr string, stop func()) {
	t.Helper()

	// Use :0 for dynamic ports
	httpSrv := gofrhttp.NewServer(gofrhttp.Address(":0"))
	grpcSrv := gofrgrpc.NewServer(":0")
	rpcxSrv := rpcx.NewServer(":0")

	srv := &service.GreeterSrv{}
	pb.RegisterHTTPServer(httpSrv, srv)
	pb.RegisterGRPCServer(grpcSrv, srv)
	pb.RegisterRPCXServer(rpcxSrv, srv)

	ctx, cancel := context.WithCancel(context.Background())
	app := gofr.New(
		gofr.Name("test"),
		gofr.Context(ctx),
		gofr.Server(httpSrv, grpcSrv, rpcxSrv),
	)

	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)

	// Extract actual addresses
	httpAddr = httpSrv.Addr()
	grpcAddr = grpcSrv.Addr()
	rpcxAddr = rpcxSrv.Addr()

	return httpAddr, grpcAddr, rpcxAddr, func() {
		cancel()
		app.Stop()
		<-done
	}
}

func TestHTTPEndpoint(t *testing.T) {
	httpAddr, _, _, stop := startApp(t)
	defer stop()

	url := fmt.Sprintf("http://%s/api/v1/greeter", httpAddr)
	body := bytes.NewReader([]byte(`{"name":"World"}`))
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var reply pb.HelloReply
	if err := json.Unmarshal(data, &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestGRPCEndpoint(t *testing.T) {
	_, grpcAddr, _, stop := startApp(t)
	defer stop()

	conn, err := gogrpc.Dial(grpcAddr,
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
		gogrpc.WithDefaultCallOptions(gogrpc.CallContentSubtype("json")),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reply := &pb.HelloReply{}
	if err := conn.Invoke(context.Background(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "gRPC"}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello gRPC" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello gRPC")
	}
}

func TestRPCXEndpoint(t *testing.T) {
	_, _, rpcxAddr, stop := startApp(t)
	defer stop()

	d, err := rpcxClient.NewPeer2PeerDiscovery("tcp@"+rpcxAddr, "")
	if err != nil {
		t.Fatal(err)
	}
	c := rpcxClient.NewOneClient(rpcxClient.Failtry, rpcxClient.RandomSelect, d, rpcxClient.DefaultOption)
	defer c.Close()

	reply := &pb.HelloReply{}
	if err := c.Call(context.Background(), "helloworld.Greeter", "SayHello", &pb.HelloRequest{Name: "rpcx"}, reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello rpcx" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello rpcx")
	}
}
