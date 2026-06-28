{{range $svc := .Services}}
func New{{$svc.ServiceType}}HTTPClient(baseURL string, hc *http.Client) *{{$svc.ServiceType}}Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &{{$svc.ServiceType}}Client{
		{{range $svc.Methods -}}
		{{.FieldName}}: func(ctx context.Context, req *{{.Request}}) (reply *{{.Reply}}, err error) {
			{{- if .HasBody}}
			var buf bytes.Buffer
			if err = json.NewEncoder(&buf).Encode(req); err != nil {
				return
			}
			httpReq, err := http.NewRequestWithContext(ctx, "{{.HTTPMethod}}", {{.HTTPURL}}, &buf)
			if err != nil {
				return
			}
			httpReq.Header.Set("Content-Type", "application/json")
			{{- else}}
			httpReq, err := http.NewRequestWithContext(ctx, "{{.HTTPMethod}}", {{.HTTPURL}}, nil)
			if err != nil {
				return
			}
			{{- end}}

			httpResp, err := hc.Do(httpReq)
			if err != nil {
				return
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode >= http.StatusBadRequest {
				err = fmt.Errorf("{{$svc.ServiceName}}.{{.Name}}: HTTP %d", httpResp.StatusCode)
				return
			}

			reply = new({{.Reply}})
			err = json.NewDecoder(httpResp.Body).Decode(reply)
			return
		},
		{{- end}}
	}
}
{{end}}
