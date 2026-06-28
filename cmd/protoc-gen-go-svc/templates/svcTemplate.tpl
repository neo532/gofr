{{range .Services}}
// {{.ServiceType}} is the server API for {{.ServiceName}} service.
type {{.ServiceType}} interface {
	{{- range .Methods}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
	{{- end}}
}

// {{.ServiceType}}Desc is the protocol-agnostic service descriptor.
var {{.ServiceType}}Desc = &transport.ServiceDesc{
	Name: "{{.ServiceName}}",
	Methods: []transport.MethodDesc{
		{{- range .Methods}}
		{
			Name: "{{.Name}}",
			NewRequest: func() any { return &{{.Request}}{} },
			{{- if .HTTPMethod }}
			HTTPMethod: "{{.HTTPMethod}}",
			HTTPPath:   "{{.HTTPPath}}",
			{{- end }}
		},
		{{- end}}
	},
}
{{end}}
