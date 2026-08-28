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

//go:embed templates/clientSetTemplate.tpl
var clientSetTmplContent string

func generateClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("client", clientTmplContent, &fileDesc{PackageName: pkg, Services: services})
}

func generateHTTPClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("http-client", httpTmplContent, &fileDesc{PackageName: pkg, Services: services})
}

func generateGRPCClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("grpc-client", grpcTmplContent, &fileDesc{PackageName: pkg, Services: services})
}

func generateRPCXClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("rpcx-client", rpcxTmplContent, &fileDesc{PackageName: pkg, Services: services})
}

func generateWSClient(pkg string, services []*serviceDesc) string {
	return renderTemplate("ws-client", wsTmplContent, &fileDesc{PackageName: pkg, Services: services})
}

func generateClientSet(data *fileDesc) string {
	return renderTemplate("client-set", clientSetTmplContent, data)
}

func renderTemplate(name, tmpl string, data *fileDesc) string {
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
