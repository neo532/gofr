package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

//go:embed svcTemplate.tpl
var tmplContent string

//go:embed httpTemplate.tpl
var httpTmplContent string

//go:embed grpcTemplate.tpl
var grpcTmplContent string

//go:embed rpcxTemplate.tpl
var rpcxTmplContent string

type paramBinding struct {
	ProtoName string // "name"
	GoField   string // "Name"
}

type methodDesc struct {
	Name       string
	Request    string
	Reply      string
	HTTPMethod string
	HTTPPath   string
	RouterPath string        // {param} → :param for httprouter
	PathParams []paramBinding // [{ProtoName:"name", GoField:"Name"}]
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

// openapiField describes a protobuf message field for OpenAPI schema generation.
type openapiField struct {
	Name     string // JSON/proto field name
	Type     string // OpenAPI type: string, integer, boolean, number
	Format   string // OpenAPI format: int64, double, float, byte
	Ref      string // $ref definition key for message types (empty for scalars)
	Repeated bool   // true if the field is repeated (array)
}

// openapi types for Swagger 2.0 spec generation.
type oasSchema struct {
	Type       string                `json:"type,omitempty"`
	Format     string                `json:"format,omitempty"`
	Ref        string                `json:"$ref,omitempty"`
	Properties map[string]*oasSchema `json:"properties,omitempty"`
	Items      *oasSchema            `json:"items,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Description string               `json:"description,omitempty"`
}

type oasParam struct {
	Name        string     `json:"name"`
	In          string     `json:"in"`
	Required    bool       `json:"required"`
	Type        string     `json:"type,omitempty"`
	Format      string     `json:"format,omitempty"`
	Schema      *oasSchema `json:"schema,omitempty"`
	Description string     `json:"description,omitempty"`
}

type oasResp struct {
	Description string     `json:"description"`
	Schema      *oasSchema `json:"schema,omitempty"`
}

type oasOp struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Parameters  []oasParam          `json:"parameters,omitempty"`
	Responses   map[string]*oasResp `json:"responses"`
}

// generateOpenAPI builds a Swagger 2.0 spec from service descriptors.
func generateOpenAPI(pkg string, services []*serviceDesc, messageDefs map[string][]openapiField) string {
	paths := make(map[string]map[string]*oasOp)
	defs := make(map[string]*oasSchema)

	// Build paths from service methods
	for _, svc := range services {
		for _, m := range svc.Methods {
			if m.HTTPMethod == "" {
				continue
			}

			operation := &oasOp{
				OperationID: svc.ServiceType + "_" + m.Name,
				Summary:     m.Name,
				Tags:        []string{svc.ServiceType},
				Responses: map[string]*oasResp{
					"200": {
						Description: "A successful response.",
						Schema:      messageSchema(m.Reply, messageDefs),
					},
				},
			}

			// Build parameters
			switch m.HTTPMethod {
			case "GET", "DELETE", "HEAD":
				for _, pp := range m.PathParams {
					operation.Parameters = append(operation.Parameters, oasParam{
						Name:     pp.ProtoName,
						In:       "path",
						Required: true,
						Type:     "string",
					})
				}
				// Query params from request message fields
				if fields, ok := messageDefs[m.Request]; ok {
					for _, f := range fields {
						if !isPathParam(f.Name, m.PathParams) {
							p := oasParam{
								Name: f.Name,
								In:   "query",
								Type: f.Type,
							}
							if f.Format != "" {
								p.Format = f.Format
							}
							operation.Parameters = append(operation.Parameters, p)
						}
					}
				}
			case "POST", "PUT", "PATCH":
				for _, pp := range m.PathParams {
					operation.Parameters = append(operation.Parameters, oasParam{
						Name:     pp.ProtoName,
						In:       "path",
						Required: true,
						Type:     "string",
					})
				}
				reqSchema := messageSchema(m.Request, messageDefs)
				if reqSchema != nil {
					operation.Parameters = append(operation.Parameters, oasParam{
						Name:     "body",
						In:       "body",
						Required: true,
						Schema:   reqSchema,
					})
				}
			}

			httpPath := m.HTTPPath
			if paths[httpPath] == nil {
				paths[httpPath] = make(map[string]*oasOp)
			}
			paths[httpPath][strings.ToLower(m.HTTPMethod)] = operation
		}
	}

	// Build definitions
	msgKeys := make([]string, 0, len(messageDefs))
	for key := range messageDefs {
		msgKeys = append(msgKeys, key)
	}
	sort.Strings(msgKeys)
	for _, key := range msgKeys {
		fields := messageDefs[key]
		s := &oasSchema{
			Type:       "object",
			Properties: make(map[string]*oasSchema),
		}
		for _, f := range fields {
			prop := fieldSchema(f, messageDefs)
			s.Properties[f.Name] = prop
		}
		defs[key] = s
	}

	spec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"title":   fmt.Sprintf("%s API", pkg),
			"version": "1.0.0",
		},
		"basePath": "/",
		"tags":     []map[string]string{},
		"schemes":  []string{"http", "https"},
		"consumes": []string{"application/json"},
		"produces": []string{"application/json"},
		"paths":    paths,
	}

	if len(defs) > 0 {
		spec["definitions"] = defs
	}

	for _, svc := range services {
		spec["tags"] = append(spec["tags"].([]map[string]string), map[string]string{"name": svc.ServiceType})
	}

	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		panic("openapi marshal: " + err.Error())
	}
	return string(b)
}

// messageSchema returns the schema for a message type.
func messageSchema(typeName string, defs map[string][]openapiField) *oasSchema {
	for key, fields := range defs {
		if strings.HasSuffix(key, "."+typeName) || key == typeName {
			s := &oasSchema{
				Type:       "object",
				Properties: make(map[string]*oasSchema),
			}
			for _, f := range fields {
				s.Properties[f.Name] = fieldSchema(f, defs)
			}
			return s
		}
	}
	return nil
}

// fieldSchema returns the schema for a single field.
func fieldSchema(f openapiField, defs map[string][]openapiField) *oasSchema {
	var s *oasSchema
	if f.Ref != "" {
		refKey := f.Ref
		if _, ok := defs[refKey]; !ok {
			for key := range defs {
				if strings.HasSuffix(key, "."+f.Ref) || key == f.Ref {
					refKey = key
					break
				}
			}
		}
		s = &oasSchema{Ref: "#/definitions/" + refKey}
	} else {
		s = &oasSchema{Type: f.Type}
		if f.Format != "" {
			s.Format = f.Format
		}
	}
	if f.Repeated {
		return &oasSchema{
			Type:  "array",
			Items: s,
		}
	}
	return s
}

// isPathParam checks if a field name appears in the path parameters list.
func isPathParam(name string, params []paramBinding) bool {
	for _, p := range params {
		if p.ProtoName == name {
			return true
		}
	}
	return false
}
