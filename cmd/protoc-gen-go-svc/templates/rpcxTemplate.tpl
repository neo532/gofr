{{range $svc := .Services}}
// {{$svc.ServiceType}}RPCXWrapper adapts {{$svc.ServiceType}} to rpcx's expected method signature.
type {{$svc.ServiceType}}RPCXWrapper struct {
	inner {{$svc.ServiceType}}
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

func _register{{$svc.ServiceType}}RPCX(s *rpcx.Server, svr {{$svc.ServiceType}}) {
	rpcx.RegisterServiceWith(s, "{{$svc.ServiceName}}", &{{$svc.ServiceType}}RPCXWrapper{inner: svr})
}
{{end}}

// RegisterRPCXServer registers all services to the rpcx server.
// It matches each svr to its service descriptor by the concrete type name.
// Panics if any defined service is left unimplemented.
func RegisterRPCXServer(s *rpcx.Server, svrs ...any) {
	matched := map[string]bool{
		{{- range $svc := .Services}}
		"{{$svc.ServiceType}}": false,
		{{- end}}
	}
	for _, svr := range svrs {
		t := reflect.TypeOf(svr)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		typeName := t.Name()
		switch typeName {
		{{- range $svc := .Services}}
		case "{{$svc.ServiceType}}":
			matched["{{$svc.ServiceType}}"] = true
			_register{{$svc.ServiceType}}RPCX(s, svr.({{$svc.ServiceType}}))
		{{- end}}
		default:
			fallbackRegisterRPCX(s, svr, matched)
		}
	}
	for svc, ok := range matched {
		if !ok {
			panic("gofr: rpcx service \"" + svc + "\" has no implementation")
		}
	}
}

func fallbackRegisterRPCX(s *rpcx.Server, svr any, matched map[string]bool) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}:
		matched["{{$svc.ServiceType}}"] = true
		_register{{$svc.ServiceType}}RPCX(s, v)
	{{- end}}
	}
}
