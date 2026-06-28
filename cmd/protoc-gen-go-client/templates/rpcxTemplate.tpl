{{range $svc := .Services}}
func New{{$svc.ServiceType}}RPCXClient(xc client.XClient) *{{$svc.ServiceType}}Client {
	return &{{$svc.ServiceType}}Client{
		{{range $svc.Methods -}}
		{{.FieldName}}: func(ctx context.Context, req *{{.Request}}) (reply *{{.Reply}}, err error) {
			reply = new({{.Reply}})
			err = xc.Call(ctx, "{{.Name}}", req, reply)
			return
		},
		{{end}}
	}
}
{{end}}
