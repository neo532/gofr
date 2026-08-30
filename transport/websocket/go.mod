module github.com/neo532/gofr/transport/websocket

go 1.25.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/julienschmidt/httprouter v1.3.0
	github.com/neo532/gofr v0.0.0
)

require github.com/neo532/gokit v1.0.48 // indirect

replace github.com/neo532/gofr => ../../

replace github.com/neo532/gokit => ../../../gokit
