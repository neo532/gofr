{{- range $svc := .Services}}
func _register{{$svc.ServiceType}}HTTP(s *http.Server, svr {{$svc.ServiceType}}) {
	{{- range $m := $svc.Methods}}
	http.HandleUnary(s, "{{$m.HTTPMethod}}", "{{$m.RouterPath}}", svr.{{$m.Name}}, {{if $m.PathParams}}func(ps httprouter.Params, req *{{$m.Request}}) error {
		{{- if $m.NeedsErrVar}}
		var err error
		{{- end}}
		{{- range $p := $m.PathParams}}
		{{$p.BindStmt}}
		{{- end}}
		return nil
	}{{else}}nil{{end}})
	{{- end}}
}
{{end}}

// RegisterHTTPServer registers all services to the HTTP server.
// It matches each svr to its service descriptor by the concrete type name.
// Panics if any defined service is left unimplemented.
func RegisterHTTPServer(s *http.Server, svrs ...any) {
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
			_register{{$svc.ServiceType}}HTTP(s, svr.({{$svc.ServiceType}}))
		{{- end}}
		default:
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
func fallbackRegisterHTTPServer(s *http.Server, svr any, matched map[string]bool) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}:
		matched["{{$svc.ServiceType}}"] = true
		_register{{$svc.ServiceType}}HTTP(s, v)
	{{- end}}
	}
}
