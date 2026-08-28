{{range $svc := .Services}}
// {{$svc.ServiceType}}ServiceName is the fully-qualified name of the {{$svc.ServiceType}} service.
const {{$svc.ServiceType}}ServiceName = "{{$svc.ServiceName}}"

func New{{$svc.ServiceType}}Client(kind string, handle any, baseURL string) *{{$svc.ServiceType}}Client {
	switch kind {
	{{- if $.HasGRPC}}
	case "grpc":
		return New{{$svc.ServiceType}}GRPCClient(handle.(grpc.ClientConnInterface))
	{{- end}}
	{{- if $.HasRPCX}}
	case "rpcx":
		return New{{$svc.ServiceType}}RPCXClient(handle.(map[string]client.XClient)[{{$svc.ServiceType}}ServiceName])
	{{- end}}
	{{- if $.HasHTTP}}
	case "http":
		return New{{$svc.ServiceType}}HTTPClient(baseURL, handle.(*http.Client))
	{{- end}}
	{{- if $.HasWS}}
	case "ws":
		return New{{$svc.ServiceType}}WSClient(baseURL, handle.(WSDialer))
	{{- end}}
	}
	return nil
}
{{end}}
