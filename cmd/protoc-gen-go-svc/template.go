package main

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

//go:embed templates/svcTemplate.tpl
var tmplContent string

//go:embed templates/httpTemplate.tpl
var httpTmplContent string

//go:embed templates/grpcTemplate.tpl
var grpcTmplContent string

//go:embed templates/rpcxTemplate.tpl
var rpcxTmplContent string

//go:embed templates/wsTemplate.tpl
var wsTmplContent string

func generate(pkg string, services []*serviceDesc) string {
	return renderTemplate("svc", tmplContent, pkg, services)
}

func generateHTTP(pkg string, services []*serviceDesc) string {
	return renderTemplate("http", httpTmplContent, pkg, services)
}

func generateGRPC(pkg string, services []*serviceDesc) string {
	return renderTemplate("grpc", grpcTmplContent, pkg, services)
}

func generateRPCX(pkg string, services []*serviceDesc) string {
	return renderTemplate("rpcx", rpcxTmplContent, pkg, services)
}

func generateWebSocket(pkg string, services []*serviceDesc) string {
	return renderTemplate("ws", wsTmplContent, pkg, services)
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
