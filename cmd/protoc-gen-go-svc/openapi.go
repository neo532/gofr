package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// generateOpenAPI builds a Swagger 2.0 spec from service descriptors.
func generateOpenAPI(pkg string, services []*serviceDesc, messageDefs map[string][]openapiField) string {
	paths := make(map[string]map[string]*oasOp)
	defs := make(map[string]*oasSchema)

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

			switch m.HTTPMethod {
			case "GET", "DELETE", "HEAD":
				for _, pp := range m.PathParams {
					operation.Parameters = append(operation.Parameters, oasParam{
						Name: pp.ProtoName, In: "path", Required: true, Type: "string",
					})
				}
				if fields, ok := messageDefs[m.Request]; ok {
					for _, f := range fields {
						if !isPathParam(f.Name, m.PathParams) {
							p := oasParam{Name: f.Name, In: "query", Type: f.Type}
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
						Name: pp.ProtoName, In: "path", Required: true, Type: "string",
					})
				}
				if reqSchema := messageSchema(m.Request, messageDefs); reqSchema != nil {
					operation.Parameters = append(operation.Parameters, oasParam{
						Name: "body", In: "body", Required: true, Schema: reqSchema,
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
			s.Properties[f.Name] = fieldSchema(f, messageDefs)
		}
		defs[key] = s
	}

	spec := map[string]any{
		"swagger": "2.0",
		"info": map[string]any{
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
		return &oasSchema{Type: "array", Items: s}
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
