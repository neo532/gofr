package main

import (
	"context"
	"fmt"
	"log"

	"github.com/neo532/gofr/core"
	pb "github.com/neo532/gofr/example/helloworld/api"
	"github.com/neo532/gofr/example/helloworld/service"
	"github.com/neo532/gofr/transport/grpc"
	"github.com/neo532/gofr/transport/http"
	"github.com/neo532/gofr/transport/rpcx"
	gofrws "github.com/neo532/gofr/transport/websocket"
	gws "github.com/gorilla/websocket"
)

func main() {
	srv := &service.GreeterSrv{}

	// ——— HTTP Server ———
	httpSrv := http.NewServer(http.Address(":8000"))
	pb.RegisterHTTPServer(httpSrv, srv)
	httpSrv.GET("/hello/:Name", func(ctx http.Context) error {
		return ctx.String(200, "custom: "+ctx.PathValue("Name"))
	})

	// ——— gRPC Server ———
	grpcSrv := grpc.NewServer(":9000")
	pb.RegisterGRPCServer(grpcSrv, srv)

	// ——— rpcx Server ———
	rpcxSrv := rpcx.NewServer(":10000")
	pb.RegisterRPCXServer(rpcxSrv, srv)

	// ——— WebSocket Server ———
	wsSrv := gofrws.NewServer(":11000")
	wsSrv.Handle("/echo", func(ctx context.Context, conn *gws.Conn) error {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if err := conn.WriteMessage(gws.TextMessage, msg); err != nil {
				break
			}
		}
		return nil
	})

	// ——— App lifecycle management ———
	app := core.New(
		core.Name("helloworld"),
		core.Version("1.0.0"),
		core.WithServer(httpSrv, grpcSrv, rpcxSrv, wsSrv),
	)
	fmt.Println("servers starting on :8000 (HTTP), :9000 (gRPC), :10000 (rpcx), :11000 (WS)...")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
