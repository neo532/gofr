{{range $svc := .Services}}
// {{$svc.ServiceType}}Client implements {{$svc.ServiceType}} via per-method function dispatch.
// Each transport constructor sets the function fields to wire up its protocol.
type {{$svc.ServiceType}}Client struct {
	{{- range $svc.Methods}}
	{{.FieldName}} func(ctx context.Context, req *{{.Request}}) (reply *{{.Reply}}, err error)
	{{- end}}
}

{{range $svc.Methods}}
func (c *{{$svc.ServiceType}}Client) {{.Name}}(ctx context.Context, req *{{.Request}}) (reply *{{.Reply}}, err error) {
	return c.{{.FieldName}}(ctx, req)
}
{{end}}
{{end}}
