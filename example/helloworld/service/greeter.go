package service

import (
	"context"

	pb "github.com/neo532/gofr/example/helloworld/api"
)

// GreeterSrv implements GreeterServer.
type GreeterSrv struct{}

// SayHello says hello — single implementation, works for all transports.
func (s *GreeterSrv) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}

func (s *GreeterSrv) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}
