{{range $svc := .Services}}
func New{{$svc.ServiceType}}WSClient(baseURL string, dialer *websocket.Dialer) *{{$svc.ServiceType}}Client {
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	return &{{$svc.ServiceType}}Client{
		{{range $svc.Methods -}}
		{{.FieldName}}: func(ctx context.Context, req *{{.Request}}) (reply *{{.Reply}}, err error) {
			conn, _, err := dialer.DialContext(ctx, baseURL+"/{{$svc.ServiceName}}/{{.Name}}", nil)
			if err != nil {
				return
			}
			defer conn.Close()

			data, err := proto.Marshal(req)
			if err != nil {
				return
			}

			if err = conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}

			_, data, err = conn.ReadMessage()
			if err != nil {
				return
			}

			reply = new({{.Reply}})
			err = proto.Unmarshal(data, reply)
			return
		},
		{{end}}
	}
}
{{end}}
