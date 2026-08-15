package main

import (
	"strings"
	"testing"
)

func TestGenerateWebSocketUsesAnnotationPath(t *testing.T) {
	// A method annotated { get: "/ws/chat" } must register the WebSocket
	// handler at that path, not at the synthesized /{ServiceName}/{Method}.
	svcs := []*serviceDesc{{
		ServiceType: "ChatService",
		ServiceName: "chat.ChatService",
		Methods: []methodDesc{{
			Name:       "Chat",
			Request:    "ChatRequest",
			HTTPMethod: "GET",
			HTTPPath:   "/ws/chat",
			RouterPath: "/ws/chat",
		}},
	}}
	out := generateWebSocket("chat", svcs)
	want := `s.Handle("GET", "/ws/chat", func(ctx context.Context, conn *websocket.Conn) error {`
	if !strings.Contains(out, want) {
		t.Fatalf("generated WS code does not register at the annotation method+path:\n%s", out)
	}
	if strings.Contains(out, `s.Handle("GET", "/chat.ChatService/Chat"`) {
		t.Fatalf("generated WS code still uses the synthesized path:\n%s", out)
	}
}

func TestGenerateWebSocketParamPath(t *testing.T) {
	// { get: "/room/{id}" } must register as the httprouter :id form so the
	// :param route on the server actually matches a request path.
	svcs := []*serviceDesc{{
		ServiceType: "ChatService",
		ServiceName: "chat.ChatService",
		Methods: []methodDesc{{
			Name:       "Chat",
			Request:    "ChatRequest",
			HTTPMethod: "GET",
			HTTPPath:   "/room/{id}",
			RouterPath: "/room/:id",
		}},
	}}
	out := generateWebSocket("chat", svcs)
	want := `s.Handle("GET", "/room/:id", func(ctx context.Context, conn *websocket.Conn) error {`
	if !strings.Contains(out, want) {
		t.Fatalf("generated WS code does not use the httprouter path form:\n%s", out)
	}
}

func TestGenerateWebSocketMethod(t *testing.T) {
	// The annotation verb is carried into the WS registration so two methods
	// sharing a path (get + put on "/user/{userId}") register without colliding.
	svcs := []*serviceDesc{{
		ServiceType: "UserApi",
		ServiceName: "user.api.user.UserApi",
		Methods: []methodDesc{
			{Name: "GetByUserId", Request: "ByIdRequest", HTTPMethod: "GET", HTTPPath: "/user/{userId}", RouterPath: "/user/:userId"},
			{Name: "PutByUserId", Request: "User", HTTPMethod: "PUT", HTTPPath: "/user/{userId}", RouterPath: "/user/:userId"},
		},
	}}
	out := generateWebSocket("user", svcs)
	if !strings.Contains(out, `s.Handle("GET", "/user/:userId", func(ctx context.Context, conn *websocket.Conn) error {`) {
		t.Fatalf("missing GET route:\n%s", out)
	}
	if !strings.Contains(out, `s.Handle("PUT", "/user/:userId", func(ctx context.Context, conn *websocket.Conn) error {`) {
		t.Fatalf("missing PUT route:\n%s", out)
	}
}
