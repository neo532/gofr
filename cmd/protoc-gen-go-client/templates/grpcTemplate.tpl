{{range $svc := .Services}}
func New{{$svc.ServiceType}}GRPCClient(conn grpc.ClientConnInterface) *{{$svc.ServiceType}}Client {
	return &{{$svc.ServiceType}}Client{
		{{range $svc.Methods -}}
		{{.FieldName}}: func(ctx context.Context, req *{{.Request}}) (reply *{{.Reply}}, err error) {
			reply = new({{.Reply}})
			err = conn.Invoke(ctx, "/{{$svc.ServiceName}}/{{.Name}}", req, reply)
			return
		},
		{{end}}
	}
}
{{end}}
