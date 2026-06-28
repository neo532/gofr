package main

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

//go:embed templates/clientTemplate.tpl
var clientTmplContent string

//go:embed templates/httpTemplate.tpl
var httpTmplContent string

//go:embed templates/grpcTemplate.tpl
var grpcTmplContent string

//go:embed templates/rpcxTemplate.tpl
var rpcxTmplContent string

//go:embed templates/wsTemplate.tpl
var wsTmplContent string

func generateClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("client", clientTmplContent, pkg, services)
}

func generateHTTPClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("http-client", httpTmplContent, pkg, services)
}

func generateGRPCClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("grpc-client", grpcTmplContent, pkg, services)
}

func generateRPCXClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("rpcx-client", rpcxTmplContent, pkg, services)
}

func generateWSClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("ws-client", wsTmplContent, pkg, services)
}

func renderTemplate(name, tmpl string, pkg string, services []*serviceDesc) string {
	data := &fileDesc{
		PackageName: pkg,
		Services:    services,
	}
	t, err := template.New(name).Parse(strings.TrimSpace(tmpl))
	if err != nil {
		panic(err)
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		panic(err)
	}
	return strings.Trim(buf.String(), "\r\n")
}
