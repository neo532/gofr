package http

import (
	"context"
	"reflect"
	"testing"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

type benchReq struct {
	Name string `json:"name"`
}

type benchReply struct {
	Message string `json:"message"`
}

type benchService struct{}

func (s *benchService) Hello(ctx context.Context, req *benchReq) (*benchReply, error) {
	return &benchReply{Message: "hello"}, nil
}

var benchSvc = &benchService{}

// BenchmarkDirectCall — bare function call baseline
func BenchmarkDirectCall(b *testing.B) {
	ctx := context.Background()
	req := &benchReq{Name: "test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSvc.Hello(ctx, req)
	}
}

// BenchmarkGenericDispatch — gofr generated code direct dispatch (0 reflection)
func BenchmarkGenericDispatch(b *testing.B) {
	ctx := context.Background()
	req := &benchReq{Name: "test"}

	var handler transport.Handler = func(ctx context.Context, req interface{}) (interface{}, error) {
		return benchSvc.Hello(ctx, req.(*benchReq))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(ctx, req)
	}
}

// BenchmarkReflectDispatch — reflect dispatch (reflect.MethodByName + Call)
func BenchmarkReflectDispatch(b *testing.B) {
	val := reflect.ValueOf(benchSvc)
	method := val.MethodByName("Hello")
	fastPath := buildFastHandler(method)

	ctx := context.Background()
	req := &benchReq{Name: "test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fastPath(ctx, req)
	}
}

// BenchmarkMiddleware2 — 2 empty middleware layers (pre-chained)
func BenchmarkMiddleware2(b *testing.B) {
	ctx := context.Background()
	req := &benchReq{Name: "test"}
	mid := middleware.Chain(
		func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req interface{}) (interface{}, error) { return next(ctx, req) }
		},
		func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req interface{}) (interface{}, error) { return next(ctx, req) }
		},
	)
	var base transport.Handler = func(ctx context.Context, req interface{}) (interface{}, error) {
		return benchSvc.Hello(ctx, req.(*benchReq))
	}
	h := mid(base)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(ctx, req)
	}
}

// BenchmarkFullChain — full chain: middleware + reflect dispatch
func BenchmarkFullChain(b *testing.B) {
	val := reflect.ValueOf(benchSvc)
	method := val.MethodByName("Hello")
	fastPath := buildFastHandler(method)

	mid := middleware.Chain(
		func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req interface{}) (interface{}, error) { return next(ctx, req) }
		},
	)
	h := mid(fastPath)

	ctx := context.Background()
	req := &benchReq{Name: "test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(ctx, req)
	}
}
