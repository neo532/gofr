{{range $svc := .Services}}
func _register{{$svc.ServiceType}}GRPC(s *grpc.Server, svr {{$svc.ServiceType}}Service) {
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
// It matches each svr to its service descriptor by the concrete type name.
func RegisterGRPCServer(s *grpc.Server, svrs ...interface{}) {
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
				_register{{$svc.ServiceType}}GRPC(s, svr.({{$svc.ServiceType}}Service))
			{{- end}}
			default:
				fallbackRegisterGRPC(s, svr)
			}
		} else {
			fallbackRegisterGRPC(s, svr)
		}
	}
}

func fallbackRegisterGRPC(s *grpc.Server, svr interface{}) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}Service:
		_register{{$svc.ServiceType}}GRPC(s, v)
	{{- end}}
	}
}
