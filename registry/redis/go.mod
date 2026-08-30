module github.com/neo532/gofr/registry/redis

go 1.25.0

require (
	github.com/neo532/gofr v0.0.0-00010101000000-000000000000
	github.com/neo532/gokit v1.0.50
	github.com/redis/go-redis/v9 v9.7.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

replace github.com/neo532/gofr => ../../

replace github.com/neo532/gokit => ../../../gokit
