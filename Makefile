# gofr framework tools

MODULE := $(shell head -1 go.mod | cut -d' ' -f2)
PROTO_DIR ?= example/helloworld/api
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
THIRD_PARTY := third_party

# Build protoc-gen-go-svc plugin
.PHONY: plugin
plugin:
	go build -o $(shell go env GOPATH)/bin/protoc-gen-go-svc ./cmd/protoc-gen-go-svc

# Generate code from proto files
.PHONY: gen
gen: plugin
	protoc -I=. -I=$(THIRD_PARTY) \
	       --go_out=. --go_opt=module=$(MODULE) \
	       --go-svc_out=. --go-svc_opt=module=$(MODULE) \
	       $(PROTO_FILES)

# Generate OpenAPI v2 specification
.PHONY: openapi
openapi:
	protoc -I=. -I=$(THIRD_PARTY) \
	       --openapiv2_out=. \
	       $(PROTO_FILES)

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
