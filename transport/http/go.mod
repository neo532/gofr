module github.com/neo532/gofr/transport/http

go 1.25.0

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/neo532/gofr v0.0.0
	github.com/neo532/gokit v1.0.48
)

require google.golang.org/protobuf v1.36.11

replace github.com/neo532/gofr => ../../

replace github.com/neo532/gokit => ../../../gokit
