{{range $svc := .Services}}
func New{{$svc.ServiceType}}HTTPClient(baseURL string, hc *{{$.HTTP}}.Client) *{{$svc.ServiceType}}Client {
	if hc == nil {
		hc = {{$.HTTP}}.DefaultClient
	}
	return &{{$svc.ServiceType}}Client{
		{{range .Methods -}}
		{{.FieldName}}: func(ctx {{$.Context}}.Context, req *{{.Request}}) (reply *{{.Reply}}, err error) {
{{.HTTPCode}}
		},
		{{- end}}
	}
}
{{end}}

type {{.HelperPrefix}}HTTPError struct {
	Method string
	Status int
	Body   []byte
}

func (e *{{.HelperPrefix}}HTTPError) Error() string {
	return {{.Fmt}}.Sprintf("%s: HTTP %d: %s", e.Method, e.Status, {{.Strings}}.TrimSpace(string(e.Body)))
}

func (e *{{.HelperPrefix}}HTTPError) HTTPStatusCode() int { return e.Status }
func (e *{{.HelperPrefix}}HTTPError) HTTPBody() []byte     { return e.Body }
