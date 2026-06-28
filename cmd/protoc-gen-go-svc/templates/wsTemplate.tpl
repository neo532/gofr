{{range $svc := .Services}}
func _register{{$svc.ServiceType}}WebSocket(s *websocket.Server, svr {{$svc.ServiceType}}) {
	{{range $svc.Methods}}
	s.Handle("/{{$svc.ServiceName}}/{{.Name}}", func(ctx context.Context, conn *websocket.Conn) error {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		req := &{{.Request}}{}
		if err := proto.Unmarshal(data, req); err != nil {
			return err
		}
		resp, err := svr.{{.Name}}(ctx, req)
		if err != nil {
			return err
		}
		data, err = proto.Marshal(resp)
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.BinaryMessage, data)
	})
	{{end}}
}
{{end}}

// RegisterWebsocketServer registers each service method as a WebSocket handler.
// It matches each svr to its service descriptor by the concrete type name.
// Panics if any defined service is left unimplemented.
func RegisterWebsocketServer(s *websocket.Server, svrs ...any) {
	matched := map[string]bool{
		{{- range $svc := .Services}}
		"{{$svc.ServiceType}}": false,
		{{- end}}
	}
	for _, svr := range svrs {
		t := reflect.TypeOf(svr)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		typeName := t.Name()
		switch typeName {
		{{- range $svc := .Services}}
		case "{{$svc.ServiceType}}":
			matched["{{$svc.ServiceType}}"] = true
			_register{{$svc.ServiceType}}WebSocket(s, svr.({{$svc.ServiceType}}))
		{{- end}}
		default:
			fallbackRegisterWebSocket(s, svr, matched)
		}
	}
	for svc, ok := range matched {
		if !ok {
			panic("gofr: websocket service \"" + svc + "\" has no implementation")
		}
	}
}

func fallbackRegisterWebSocket(s *websocket.Server, svr any, matched map[string]bool) {
	switch v := svr.(type) {
	{{- range $svc := .Services}}
	case {{$svc.ServiceType}}:
		matched["{{$svc.ServiceType}}"] = true
		_register{{$svc.ServiceType}}WebSocket(s, v)
	{{- end}}
	}
}
