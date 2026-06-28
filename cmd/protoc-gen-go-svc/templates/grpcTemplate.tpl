{{range $svc := .Services}}
func _register{{$svc.ServiceType}}GRPC(s *grpc.Server, svr {{$svc.ServiceType}}) {
	grpc.RegisterServiceWith(s, "{{$svc.ServiceName}}", svr, []struct {
		Name    string
		NewReq  func() any
		Handler grpc.UnaryHandler
	}{
		{{- range $svc.Methods}}
		{Name: "{{.Name}}", NewReq: func() any { return &{{.Request}}{} }, Handler: func(ctx context.Context, req any) (any, error) {
			return svr.{{.Name}}(ctx, req.(*{{.Request}}))
		}},
		{{- end}}
	})
}
{{end}}

// RegisterGRPCServer registers all services to the gRPC server.
// It matches each svr to its service descriptor by the concrete type name.
// Panics if any defined service is left unimplemented.
func RegisterGRPCServer(s *grpc.Server, svrs ...any) {
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
			_register{{$svc.ServiceType}}GRPC(s, svr.({{$svc.ServiceType}}))
		{{- end}}
		default:
			fallbackRegisterGRPC(s, svr, matched)
		}
	}
	for svc, ok := range matched {
		if !ok {
			panic("gofr: gRPC service \"" + svc + "\" has no implementation")
		}
	}
}

func fallbackRegisterGRPC(s *grpc.Server, svr any, matched map[string]bool) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}:
		matched["{{$svc.ServiceType}}"] = true
		_register{{$svc.ServiceType}}GRPC(s, v)
	{{- end}}
	}
}
