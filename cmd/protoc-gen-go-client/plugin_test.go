package main

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestBuildHTTPMethodCodeUsesDirectFields(t *testing.T) {
	input := testMessage(t, "Request", []*descriptorpb.FieldDescriptorProto{
		{Name: proto.String("app_token"), JsonName: proto.String("appToken"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
		{Name: proto.String("page_size"), JsonName: proto.String("pageSize"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
		{Name: proto.String("body"), JsonName: proto.String("body"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".test.Body")},
	})
	body := testMessage(t, "Body", nil)
	input.Fields[2].Message = body
	method := &protogen.Method{GoName: "Create", Input: input, Output: &protogen.Message{GoIdent: protogen.GoIdent{GoName: "Reply"}}}

	code, reqs, err := buildHTTPMethodCode("test.Api", method, "POST", "/apps/{app_token}", "body", "testHTTP")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"req.AppToken", "req.PageSize", "req.Body", "strconv.Itoa", "protojson.MarshalOptions", "url.PathEscape", "Content-Type"} {
		if !strings.Contains(code, want) {
			t.Errorf("generated code does not contain %q:\n%s", want, code)
		}
	}
	for _, unwanted := range []string{"GetAppToken", "PathValue", "JSONField", "encoding/json"} {
		if strings.Contains(code, unwanted) {
			t.Errorf("generated code contains removed runtime helper %q:\n%s", unwanted, code)
		}
	}
	if !reqs.Bytes || !reqs.URL || !reqs.Strconv {
		t.Fatalf("unexpected requirements: %+v", reqs)
	}
}

func TestBuildPathExprAddsNestedNilGuard(t *testing.T) {
	input := testMessage(t, "Request", []*descriptorpb.FieldDescriptorProto{
		{Name: proto.String("resource"), JsonName: proto.String("resource"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".test.Resource")},
	})
	resource := testMessage(t, "Resource", []*descriptorpb.FieldDescriptorProto{
		{Name: proto.String("name"), JsonName: proto.String("name"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
	})
	input.Fields[0].Message = resource

	expr, guards, _, err := buildPathExpr(input, "/v1/{resource.name}")
	if err != nil {
		t.Fatal(err)
	}
	if expr != `baseURL + "/v1/" + url.PathEscape(req.Resource.Name)` {
		t.Fatalf("unexpected path expression: %s", expr)
	}
	if len(guards) != 1 || guards[0] != "req.Resource == nil" {
		t.Fatalf("unexpected nil guards: %#v", guards)
	}
}

func TestHTTPValueExprUsesStrconv(t *testing.T) {
	field := testField(t, "count", descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL)
	value, usesStrconv, err := httpValueExpr("req.Count", field, false)
	if err != nil {
		t.Fatal(err)
	}
	if value != "strconv.FormatInt(req.Count, 10)" || !usesStrconv {
		t.Fatalf("unexpected expression %q, usesStrconv=%v", value, usesStrconv)
	}
}

func testMessage(t *testing.T, name string, fields []*descriptorpb.FieldDescriptorProto) *protogen.Message {
	t.Helper()
	messageTypes := []*descriptorpb.DescriptorProto{{Name: proto.String(name), Field: fields}}
	for _, field := range fields {
		if field.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE || field.GetTypeName() == "" {
			continue
		}
		typeName := field.GetTypeName()
		if dot := strings.LastIndexByte(typeName, '.'); dot >= 0 {
			typeName = typeName[dot+1:]
		}
		messageTypes = append(messageTypes, &descriptorpb.DescriptorProto{Name: proto.String(typeName)})
	}
	fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:      proto.String("proto3"),
		Name:        proto.String("test.proto"),
		Package:     proto.String("test"),
		MessageType: messageTypes,
	}, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatal(err)
	}
	message := fd.Messages().Get(0)
	result := &protogen.Message{Desc: message, GoIdent: protogen.GoIdent{GoName: name}}
	for i := 0; i < message.Fields().Len(); i++ {
		field := message.Fields().Get(i)
		result.Fields = append(result.Fields, &protogen.Field{Desc: field, GoName: testGoName(string(field.Name()))})
	}
	return result
}

func testGoName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
func testField(t *testing.T, name string, kind descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label) *protogen.Field {
	return testMessage(t, "Request", []*descriptorpb.FieldDescriptorProto{{
		Name: proto.String(name), JsonName: proto.String(name), Number: proto.Int32(1), Label: label.Enum(), Type: kind.Enum(),
	}}).Fields[0]
}
