// Use RegisterHTTPServer to register services.
// For per-service registration: RegisterService(s, {{(index .Services 0).ServiceType}}ServiceDesc, svr)

// RegisterHTTPServer registers all services to the HTTP server.
// It matches each svr to its service descriptor by extracting the service name
// from the concrete type: the text before "Service" + the text after "Service".
// e.g. "DemoService" -> "Demo", "Demo1Service" -> "Demo1", "DemoService1" -> "Demo1".
// Falls back to interface type assertion if extraction fails.
// Panics if any defined service is left unimplemented.
func RegisterHTTPServer(s *http.Server, svrs ...interface{}) {
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
		if idx := strings.Index(typeName, "Service"); idx >= 0 {
			svcName := typeName[:idx] + typeName[idx+len("Service"):]
			switch svcName {
			{{- range $svc := .Services}}
			case "{{$svc.ServiceType}}":
				matched["{{$svc.ServiceType}}"] = true
				http.RegisterService(s, {{$svc.ServiceType}}ServiceDesc, svr)
			{{- end}}
			default:
				fallbackRegisterHTTPServer(s, svr, matched)
			}
		} else {
			fallbackRegisterHTTPServer(s, svr, matched)
		}
	}
	for svc, ok := range matched {
		if !ok {
			panic("gofr: HTTP service \"" + svc + "\" has no implementation")
		}
	}
}

// fallbackRegisterHTTPServer tries interface-based matching.
func fallbackRegisterHTTPServer(s *http.Server, svr interface{}, matched map[string]bool) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}Service:
		matched["{{$svc.ServiceType}}"] = true
		http.RegisterService(s, {{$svc.ServiceType}}ServiceDesc, v)
	{{- end}}
	}
}
