package main

import "strings"

type methodDesc struct {
	Name      string
	FieldName string // unexported function field name, e.g. "postFn"
	Request   string
	Reply     string
	HTTPCode  string
}

type serviceDesc struct {
	ServiceType string
	ServiceName string
	Methods     []methodDesc
}

type fileDesc struct {
	PackageName string
	Services    []*serviceDesc
	HasHTTP     bool
	HasGRPC     bool
	HasRPCX     bool
	HasWS       bool

	HelperPrefix string
	Context      string
	Bytes        string
	Fmt          string
	IO           string
	HTTP         string
	URL          string
	ProtoJSON    string
	Strconv      string
	Strings      string
}

// fieldName converts a method name to an unexported function field name.
// Post → postFn, GetById → getByIdFn
func fieldName(name string) string {
	if len(name) == 0 {
		return "fn"
	}
	return strings.ToLower(name[:1]) + name[1:] + "Fn"
}
