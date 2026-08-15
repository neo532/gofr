module github.com/neo532/gofr/middleware/trace

go 1.25.0

require (
	github.com/neo532/gofr v0.0.0
	go.opentelemetry.io/otel v1.45.0
	go.opentelemetry.io/otel/trace v1.45.0
)

replace github.com/neo532/gofr => ../../

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/neo532/gokit v1.0.40 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
)
