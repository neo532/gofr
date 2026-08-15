{{range .Services}}
// {{.ServiceType}} is the server API for {{.ServiceName}} service.
type {{.ServiceType}} interface {
	{{- range .Methods}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
	{{- end}}
}
{{end}}
