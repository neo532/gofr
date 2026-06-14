{{range $svc := .Services}}
// {{$svc.ServiceType}}RPCXWrapper adapts {{$svc.ServiceType}}Service to rpcx's expected method signature.
type {{$svc.ServiceType}}RPCXWrapper struct {
	inner {{$svc.ServiceType}}Service
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

func _register{{$svc.ServiceType}}RPCX(s *rpcx.Server, svr {{$svc.ServiceType}}Service) {
	rpcx.RegisterServiceWith(s, "{{$svc.ServiceName}}", &{{$svc.ServiceType}}RPCXWrapper{inner: svr})
}
{{end}}

// RegisterRPCXServer registers all services to the rpcx server.
// It matches each svr to its service descriptor by the concrete type name.
func RegisterRPCXServer(s *rpcx.Server, svrs ...interface{}) {
	for _, svr := range svrs {
		t := reflect.TypeOf(svr)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		typeName := t.Name()
		if idx := strings.Index(typeName, "Service"); idx >= 0 {
			svcName := typeName[:idx] + typeName[idx+len("Service"):]
			switch svcName {
			{{- range $svc := .Services}}
			case "{{$svc.ServiceType}}":
				_register{{$svc.ServiceType}}RPCX(s, svr.({{$svc.ServiceType}}Service))
			{{- end}}
			default:
				fallbackRegisterRPCX(s, svr)
			}
		} else {
			fallbackRegisterRPCX(s, svr)
		}
	}
}

func fallbackRegisterRPCX(s *rpcx.Server, svr interface{}) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}Service:
		_register{{$svc.ServiceType}}RPCX(s, v)
	{{- end}}
	}
}
