package main

import (
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

// extractHTTPBinding reads google.api.http annotation from a proto method.
func extractHTTPBinding(method *protogen.Method) (httpMethod, httpPath string) {
	opts := method.Desc.Options()
	if opts == nil {
		return "", ""
	}
	rule, ok := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	if !ok || rule == nil {
		return "", ""
	}
	switch p := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return "GET", p.Get
	case *annotations.HttpRule_Post:
		return "POST", p.Post
	case *annotations.HttpRule_Put:
		return "PUT", p.Put
	case *annotations.HttpRule_Delete:
		return "DELETE", p.Delete
	case *annotations.HttpRule_Patch:
		return "PATCH", p.Patch
	}
	return "", ""
}

// toHTTPRouterPath converts {param} to :param for httprouter.
func toHTTPRouterPath(p string) string {
	var b strings.Builder
	b.Grow(len(p))
	for i := 0; i < len(p); i++ {
		if p[i] == '{' {
			b.WriteByte(':')
			i++
			for i < len(p) && p[i] != '}' {
				b.WriteByte(p[i])
				i++
			}
		} else {
			b.WriteByte(p[i])
		}
	}
	return b.String()
}

// extractPathParams extracts path parameter names and their Go field names from an HTTP path.
// e.g. "/api/v1/hello/{name}" -> [{ProtoName:"name", GoField:"Name"}]
func extractPathParams(path string) (params []paramBinding) {
	for i := 0; i < len(path); i++ {
		if path[i] == '{' {
			start := i + 1
			i++
			for i < len(path) && path[i] != '}' {
				i++
			}
			if start < i {
				name := path[start:i]
				params = append(params, paramBinding{
					ProtoName: name,
					GoField:   snakeToPascal(name),
				})
			}
		}
	}
	return
}

// snakeToPascal converts snake_case to PascalCase.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// splitLines splits a string into lines, used to write generated code line-by-line.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
