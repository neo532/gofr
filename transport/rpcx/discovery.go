package rpcx

import (
	"context"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MetadataApiName is the rpcx service name that exposes descriptor dumps so a
// gateway can translate REST URLs without linking this binary's proto module.
// Gateways reference this name as a literal string.
const MetadataApiName = "metadata.MetadataApi"

// MetadataApi dumps the transitive FileDescriptorSet closure of every proto
// service registered on the server, including google.api.http annotations.
type MetadataApi struct {
	names func() []string
}

func newMetadataApi(names func() []string) *MetadataApi {
	return &MetadataApi{names: names}
}

// Dump implements the rpcx method convention (ctx, args, reply) error.
func (m *MetadataApi) Dump(_ context.Context, _ *emptypb.Empty, reply *descriptorpb.FileDescriptorSet) error {
	seen := make(map[string]bool)
	var files []*descriptorpb.FileDescriptorProto
	for _, name := range m.names() {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
		if err != nil {
			continue // hand-written rpcx service without a proto descriptor
		}
		collectFileClosure(d.ParentFile(), seen, &files)
	}
	reply.File = files
	return nil
}

func collectFileClosure(fd protoreflect.FileDescriptor, seen map[string]bool, out *[]*descriptorpb.FileDescriptorProto) {
	path := fd.Path()
	if seen[path] {
		return
	}
	seen[path] = true
	for i := 0; i < fd.Imports().Len(); i++ {
		collectFileClosure(fd.Imports().Get(i).FileDescriptor, seen, out)
	}
	*out = append(*out, protodesc.ToFileDescriptorProto(fd))
}
