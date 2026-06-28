package main

import "strings"

type paramBinding struct {
	ProtoName string
	GoField   string
}

type methodDesc struct {
	Name       string
	FieldName  string // unexported function field name, e.g. "postFn"
	Request    string
	Reply      string
	HTTPMethod string
	HTTPPath   string
	PathParams []paramBinding
	HTTPURL    string // pre-built Go URL expression
	HasBody    bool
}

type serviceDesc struct {
	ServiceType string
	ServiceName string
	Methods     []methodDesc
}

type fileDesc struct {
	PackageName string
	Services    []*serviceDesc
}

// fieldName converts a method name to an unexported function field name.
// Post → postFn, GetById → getByIdFn
func fieldName(name string) string {
	if len(name) == 0 {
		return "fn"
	}
	return strings.ToLower(name[:1]) + name[1:] + "Fn"
}
