import "github.com/neo532/gofr/transport/http"

{{range $svc := .Services}}
func _register{{$svc.ServiceType}}HTTP(s *http.Server, svr {{$svc.ServiceType}}Server) {
	{{- range $i, $m := $svc.Methods}}
	http.RegisterUnary(s, "{{if $m.HTTPMethod}}{{$m.HTTPMethod}}{{else}}POST{{end}}", "{{$m.RouterPath}}", svr.{{$m.Name}}, func(ctx http.Context, req *{{$m.Request}}) error {
		{{- if $m.PathParams}}
		{{- range $p := $m.PathParams}}
		if v := ctx.PathValue("{{$p.ProtoName}}"); v != "" { req.{{$p.GoField}} = v }
		{{- end}}
		return nil
		{{- else}}
		return ctx.Bind(req)
		{{- end}}
	})
	{{- end}}
}
{{end}}

// RegisterHTTPServer registers all services to the HTTP server.
func RegisterHTTPServer(s *http.Server, svrs ...interface{}) {
	for _, svr := range svrs {
		switch v := svr.(type) {
		{{- range $svc := .Services}}
		case {{$svc.ServiceType}}Server:
			_register{{$svc.ServiceType}}HTTP(s, v)
		{{- end}}
		}
	}
}
