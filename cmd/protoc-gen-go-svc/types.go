package main

// paramBinding maps a path parameter name to its Go struct field.
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
	PathParams []paramBinding
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

// openapiField describes a protobuf message field for OpenAPI schema generation.
type openapiField struct {
	Name     string
	Type     string
	Format   string
	Ref      string
	Repeated bool
}

// oas types for Swagger 2.0 spec generation.
type oasSchema struct {
	Type        string               `json:"type,omitempty"`
	Format      string               `json:"format,omitempty"`
	Ref         string               `json:"$ref,omitempty"`
	Properties  map[string]*oasSchema `json:"properties,omitempty"`
	Items       *oasSchema           `json:"items,omitempty"`
	Required    []string             `json:"required,omitempty"`
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
