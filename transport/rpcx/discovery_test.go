package rpcx

import (
	"context"
	"testing"

	rpcxClient "github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func newRPCXProtoClient(t *testing.T, addr string) *rpcxClient.OneClient {
	t.Helper()
	d, err := rpcxClient.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		t.Fatal(err)
	}
	opt := rpcxClient.DefaultOption
	opt.SerializeType = protocol.ProtoBuffer
	c := rpcxClient.NewOneClient(rpcxClient.Failtry, rpcxClient.RandomSelect, d, opt)
	t.Cleanup(func() { c.Close() })
	return c
}

// dumpDummy is a valid rpcx service registered under a proto full name so the
// MetadataApi dump can resolve it against the global registry.
type dumpDummy struct{}

func (s *dumpDummy) Get(ctx context.Context, args *emptypb.Empty, reply *emptypb.Empty) error {
	return nil
}

func registerDumpEchoFile(t *testing.T) {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/api/dump_echo.proto"),
		Package:    proto.String("test.api"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/empty.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("EchoApi"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("Get"),
						InputType:  proto.String(".google.protobuf.Empty"),
						OutputType: proto.String(".google.protobuf.Empty"),
					},
				},
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("build dump_echo file: %v", err)
	}
	if _, err := protoregistry.GlobalFiles.FindFileByPath(fd.Path()); err == nil {
		return // already registered
	}
	if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
		t.Fatalf("register dump_echo file: %v", err)
	}
}

func TestMetadataApiDump(t *testing.T) {
	registerDumpEchoFile(t)

	srv := NewServer(Address(":0"))
	RegisterServiceWith(srv, "test.api.EchoApi", &dumpDummy{})
	addr, stop := newTestServer(t, srv)
	defer stop()

	c := newRPCXProtoClient(t, addr)
	reply := &descriptorpb.FileDescriptorSet{}
	if err := c.Call(context.Background(), MetadataApiName, "Dump", &emptypb.Empty{}, reply); err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool)
	for _, f := range reply.File {
		got[f.GetName()] = true
	}
	for _, want := range []string{"test/api/dump_echo.proto", "google/protobuf/empty.proto"} {
		if !got[want] {
			t.Fatalf("dump missing %s; got %v", want, got)
		}
	}
}
