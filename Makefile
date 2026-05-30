# gofr framework tools

# Build protoc-gen-go-svc plugin
.PHONY: plugin
plugin:
	go build -o $(shell go env GOPATH)/bin/protoc-gen-go-svc ./cmd/protoc-gen-go-svc

# Generate code from proto files
.PHONY: gen
gen: plugin
	protoc -I=. -I=third_party \
	       --go_out=. --go_opt=module=github.com/neo532/gofr \
	       --go-svc_out=. --go-svc_opt=module=github.com/neo532/gofr \
	       example/helloworld/api/helloworld.proto

# Generate OpenAPI v2 specification
.PHONY: openapi
openapi:
	protoc -I=. -I=third_party \
	       --openapiv2_out=. \
	       example/helloworld/api/helloworld.proto

# Generate everything (proto code + OpenAPI)
.PHONY: all
all: gen openapi

# Build all packages
.PHONY: build
build:
	go build ./...

# Run all tests
.PHONY: test
test:
	go test ./...

# Run benchmarks
.PHONY: bench
bench:
	go test ./transport/http/... -bench=. -benchmem -run=^$
