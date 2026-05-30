import (
	"context"

	"github.com/neo532/gofr/transport"
)

{{range .Services}}
// {{.ServiceType}}Server is the server API for {{.ServiceName}} service.
type {{.ServiceType}}Server interface {
	{{- range .Methods}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
	{{- end}}
}

// {{.ServiceType}}ServiceDesc is the protocol-agnostic service descriptor.
var {{.ServiceType}}ServiceDesc = &transport.ServiceDesc{
	Name: "{{.ServiceName}}",
	Methods: []transport.MethodDesc{
		{{- range .Methods}}
		{
			Name: "{{.Name}}",
			NewRequest: func() interface{} { return &{{.Request}}{} },
			{{- if .HTTPMethod }}
			HTTPMethod: "{{.HTTPMethod}}",
			HTTPPath:   "{{.HTTPPath}}",
			{{- end }}
		},
		{{- end}}
	},
}
{{end}}
