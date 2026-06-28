# gofr

**gofr** is a multi-protocol Go microservice framework. Define your service once in protobuf, generate code for all transports, share middleware across protocols.

## Why gofr?

Most Go frameworks lock you into a single transport protocol. gofr decouples the service definition from the transport layer, so you can expose the same service over multiple protocols with zero extra code — and swap or add protocols without touching business logic.

- **Protocol-agnostic service definitions** — A single `ServiceDesc` describes your RPC methods. Each transport knows how to serve it.
- **Code-generated registration** — `protoc-gen-go-svc` plugin generates transport bindings from `.proto` files. No manual wiring.
- **Zero-reflection request path** — Generics and pre-built closures eliminate reflection at request time. Reflection is used only at startup.
- **Shared middleware** — Define middleware once, apply it to HTTP, gRPC, rpcx, or WebSocket uniformly.
- **Per-protocol dependencies** — Each transport is a separate Go module. Import only what you use — no thrift leaking into HTTP-only services.

## Transports

| Protocol | Module | Underlying Library |
|---|---|---|
| HTTP/1.1 | `github.com/neo532/gofr/transport/http` | [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) |
| gRPC | `github.com/neo532/gofr/transport/grpc` | [google.golang.org/grpc](https://google.golang.org/grpc) |
| rpcx | `github.com/neo532/gofr/transport/rpcx` | [smallnest/rpcx](https://github.com/smallnest/rpcx) |
| WebSocket | `github.com/neo532/gofr/transport/websocket` | [gorilla/websocket](https://github.com/gorilla/websocket) |

## Quick Start

### 1. Define your service in proto

```protobuf
syntax = "proto3";
import "google/api/annotations.proto";

package helloworld;
option go_package = "github.com/neo532/gofr/example/helloworld/api;api";

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {
    option (google.api.http) = {
      post: "/api/v1/greeter"
      body: "*"
    };
  }
  rpc Hello (HelloRequest) returns (HelloReply) {
    option (google.api.http) = {
      get: "/api/v1/hello/{name}"
    };
  }
}

message HelloRequest { string name = 1; }
message HelloReply { string message = 1; }
```

### 2. Generate code

```sh
protoc -I=. -I=third_party \
  --go_out=. --go_opt=module=github.com/neo532/gofr \
  --go-svc_out=. --go-svc_opt=module=github.com/neo532/gofr,protocols=http,grpc,rpcx,openapi \
  api/helloworld.proto
```

This generates:
- `helloworld.pb.go` — protobuf types (via `protoc-gen-go`)
- `helloworld_svc.pb.go` — `GreeterServer` interface + `GreeterServiceDesc`
- `helloworld_http.pb.go` — HTTP registration
- `helloworld_grpc.pb.go` — gRPC registration
- `helloworld_rpcx.pb.go` — rpcx wrapper + registration
- `helloworld.swagger.json` — OpenAPI v2 spec

### 3. Implement the service

```go
package service

import (
    "context"
    pb "github.com/neo532/gofr/example/helloworld/api"
)

type GreeterSrv struct{}

func (s *GreeterSrv) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}
```

One implementation, all transports.

### 4. Start the app

```go
package main

import (
    "github.com/neo532/gofr"
    pb "github.com/neo532/gofr/example/helloworld/api"
    "github.com/neo532/gofr/example/helloworld/service"
    "github.com/neo532/gofr/transport/http"
    "github.com/neo532/gofr/transport/grpc"
)

func main() {
    srv := &service.GreeterSrv{}

    httpSrv := http.NewServer(http.Address(":8000"))
    pb.RegisterHTTPServer(httpSrv, srv)

    grpcSrv := grpc.NewServer(":9000")
    pb.RegisterGRPCServer(grpcSrv, srv)

    app := gofr.New(
        gofr.Name("helloworld"),
        gofr.Version("1.0.0"),
        gofr.Server(httpSrv, grpcSrv),
    )
    app.Run()
}
```

## Architecture

```
protoc (proto file)
  └─> protoc-gen-go-svc plugin
        ├─ _svc.pb.go     (interface + ServiceDesc)
        ├─ _http.pb.go    (HTTP registration)
        ├─ _grpc.pb.go    (gRPC registration)
        ├─ _rpcx.pb.go    (rpcx wrapper + registration)
        └─ .swagger.json  (OpenAPI spec)

Application:
  transport.Server (HTTP, gRPC, rpcx, WS)
    └─ generated Register*Server() binds your implementation
       └─ gofr.New(Name, Version, WithServer(...))
          └─ app.Run() → errgroup starts all servers ← OS signal triggers graceful shutdown
```

### Request flow (HTTP example)

```
HTTP Request → httprouter → generated handler closure
  → DecodeRequestFunc (JSON)
  → Pre-built middleware chain (global + per-method)
  → Your service method (e.g., SayHello)
  → EncodeResponseFunc (JSON)
```

### Transport context

Every request carries a `transport.Transporter` in the context, accessible via `transport.FromServerContext(ctx)`. This gives you uniform access to transport kind, endpoint, operation, and headers — regardless of which protocol handled the request.

```go
func MyMiddleware(next transport.Handler) transport.Handler {
    return func(ctx context.Context, req any) (any, error) {
        if tr, ok := transport.FromServerContext(ctx); ok {
            log.Printf("kind=%s operation=%s", tr.Kind(), tr.Operation())
        }
        return next(ctx, req)
    }
}
```

## Middleware

gofr provides a universal middleware signature:

```go
type Middleware func(transport.Handler) transport.Handler
```

Apply middleware globally or per-method:

```go
// Global — applies to every endpoint
srv := http.NewServer(http.Address(":8000"),
    http.Middleware(logging, auth),
)

// Per-method — applies only to a specific operation
srv.UseWith("/api/v1/greeter", rateLimit)
```

Middleware works identically across HTTP, gRPC, rpcx, and WebSocket.

## Lifecycle

`gofr.App` manages all servers as a unified group:

- Starts all servers concurrently via `errgroup`
- Listens for OS signals (`SIGINT`, `SIGTERM`)
- Graceful shutdown with configurable timeout (default 10s)
- Lifecycle hooks: `BeforeStart`, `AfterStart`, `BeforeStop`, `AfterStop`

## Dependency Management

Each transport is a separate Go module with its own `go.mod`. Import only what you use:

| Import | Result |
|--------|--------|
| `transport/http` | Only httprouter added |
| `transport/http` + `transport/grpc` | httprouter + grpc |
| `transport/http` + `transport/rpcx` | httprouter + rpcx + thrift |
| `transport/http` + `transport/websocket` | httprouter + gorilla/websocket |

Your application's `go.mod` only contains dependencies for the transports you actually import.

## Extending

To add a new protocol transport, follow the pattern documented in [CONTRIBUTING.md](CONTRIBUTING.md):

1. Create `transport/<name>/` with server, middleware manager, and tests
2. Add registration function
3. Create a protoc plugin template
4. Wire it into `template.go` and `main.go`
5. Regenerate the example
6. Add integration test

## Example

A complete multi-protocol example is in [`example/helloworld/`](example/helloworld/). It demonstrates:

- HTTP on `:8000` (with custom route and generated registration)
- gRPC on `:9000`
- rpcx on `:10000`
- WebSocket echo on `:11000`
- Single service implementation for all RPC protocols
- Integration tests covering all transports

Run it:

```sh
cd example/helloworld && go run ./cmd/
```
