import (
	"context"

	"github.com/neo532/gofr/transport/grpc"
)

{{range $svc := .Services}}
func _register{{$svc.ServiceType}}GRPC(s *grpc.Server, svr {{$svc.ServiceType}}Server) {
	grpc.RegisterServiceWith(s, "{{$svc.ServiceName}}", svr, []struct {
		Name    string
		NewReq  func() interface{}
		Handler grpc.UnaryHandler
	}{
		{{- range $svc.Methods}}
		{Name: "{{.Name}}", NewReq: func() interface{} { return &{{.Request}}{} }, Handler: func(ctx context.Context, req interface{}) (interface{}, error) {
			return svr.{{.Name}}(ctx, req.(*{{.Request}}))
		}},
		{{- end}}
	})
}
{{end}}

// RegisterGRPCServer registers all services to the gRPC server.
func RegisterGRPCServer(s *grpc.Server, svrs ...interface{}) {
	for _, svr := range svrs {
		switch v := svr.(type) {
		{{- range $svc := .Services}}
		case {{$svc.ServiceType}}Server:
			_register{{$svc.ServiceType}}GRPC(s, v)
		{{- end}}
		}
	}
}
