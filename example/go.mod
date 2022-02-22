module github.com/neo532/gofr/example

go 1.17

require (
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/neo532/gofr v0.0.0-00010101000000-000000000000
)

require (
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.18.1 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
)

replace github.com/neo532/gofr => ../
