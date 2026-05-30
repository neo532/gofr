import (
	"context"

	"github.com/neo532/gofr/transport/rpcx"
	"google.golang.org/protobuf/proto"
)

{{range $svc := .Services}}
// {{$svc.ServiceType}}RPCXWrapper adapts GreeterServer to rpcx's expected method signature.
type {{$svc.ServiceType}}RPCXWrapper struct {
	inner {{$svc.ServiceType}}Server
}

{{range $svc.Methods}}
func (w *{{$svc.ServiceType}}RPCXWrapper) {{.Name}}(ctx context.Context, args *{{.Request}}, reply *{{.Reply}}) error {
	result, err := w.inner.{{.Name}}(ctx, args)
	if err != nil {
		return err
	}
	proto.Merge(reply, result)
	return nil
}
{{end}}

func _register{{$svc.ServiceType}}RPCX(s *rpcx.Server, svr {{$svc.ServiceType}}Server) {
	rpcx.RegisterServiceWith(s, "{{$svc.ServiceName}}", &{{$svc.ServiceType}}RPCXWrapper{inner: svr})
}
{{end}}

// RegisterRPCXServer registers all services to the rpcx server.
func RegisterRPCXServer(s *rpcx.Server, svrs ...interface{}) {
	for _, svr := range svrs {
		switch v := svr.(type) {
		{{- range $svc := .Services}}
		case {{$svc.ServiceType}}Server:
			_register{{$svc.ServiceType}}RPCX(s, v)
		{{- end}}
		}
	}
}
