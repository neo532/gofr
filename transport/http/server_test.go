package http

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/neo532/gofr/transport"
)

type helloReq struct {
	Name string `json:"name"`
}

type helloReply struct {
	Message string `json:"message"`
}

type testService struct{}

func (s *testService) SayHello(ctx context.Context, req *helloReq) (*helloReply, error) {
	return &helloReply{Message: "Hello " + req.Name}, nil
}

var testServiceDesc = &transport.ServiceDesc{
	Name: "test.Greeter",
	Methods: []transport.MethodDesc{
		{
			Name:       "SayHello",
			NewRequest: func() interface{} { return &helloReq{} },
		},
	},
}

func startServer(t *testing.T, srv *Server) (addr string, stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)
	_, port, _ := net.SplitHostPort(srv.lis.Addr().String())
	addr = "127.0.0.1:" + port
	stop = func() {
		cancel()
		<-done
	}
	return
}

func TestRegisterService(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterService(srv, testServiceDesc, &testService{})

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Post(
		"http://"+addr+"/test.Greeter/SayHello",
		"application/json",
		strings.NewReader(`{"name":"World"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestCustomRoute(t *testing.T) {
	srv := NewServer(Address(":0"))
	srv.GET("/hello/:Name", func(ctx Context) error {
		return ctx.String(200, "hello "+ctx.PathValue("Name"))
	})

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Get("http://" + addr + "/hello/GoFr")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello GoFr" {
		t.Fatalf("got %q, want %q", string(body), "hello GoFr")
	}
}

func TestAnnotationRoute(t *testing.T) {
	desc := &transport.ServiceDesc{
		Name: "test.Greeter",
		Methods: []transport.MethodDesc{
			{
				Name:       "SayHello",
				NewRequest: func() interface{} { return &helloReq{} },
				HTTPMethod: "GET",
				HTTPPath:   "/api/v1/hello/{name}",
			},
		},
	}

	srv := NewServer(Address(":0"))
	RegisterService(srv, desc, &testService{})

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Get("http://" + addr + "/api/v1/hello/World")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestToHTTPRouterPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/api/v1/hello/{name}", "/api/v1/hello/:name"},
		{"/api/v1/{userId}/posts/{postId}", "/api/v1/:userId/posts/:postId"},
		{"/api/v1/greeter", "/api/v1/greeter"},
		{"/v1/{a}/{b}/{c}", "/v1/:a/:b/:c"},
	}
	for _, tt := range tests {
		got := toHTTPRouterPath(tt.in)
		if got != tt.want {
			t.Errorf("toHTTPRouterPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSnakeToPascal(t *testing.T) {
	tests := []struct{ in, want string }{
		{"name", "Name"},
		{"user_id", "UserId"},
		{"hello_world_test", "HelloWorldTest"},
	}
	for _, tt := range tests {
		got := snakeToPascal(tt.in)
		if got != tt.want {
			t.Errorf("snakeToPascal(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/api/v1/hello/{name}", []string{"name"}},
		{"/api/v1/{userId}/posts/{postId}", []string{"userId", "postId"}},
		{"/api/v1/greeter", nil},
	}
	for _, tt := range tests {
		got := extractPathParams(tt.path)
		if len(got) != len(tt.want) {
			t.Errorf("extractPathParams(%q) = %v, want %v", tt.path, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractPathParams(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
			}
		}
	}
}

func TestServerMiddleware(t *testing.T) {
	var logged bool

	srv := NewServer(Address(":0"),
		Middleware(func(next transport.Handler) transport.Handler {
			return func(ctx context.Context, req interface{}) (interface{}, error) {
				logged = true
				return next(ctx, req)
			}
		}),
	)
	RegisterService(srv, testServiceDesc, &testService{})

	addr, stop := startServer(t, srv)
	defer stop()

	http.Post("http://"+addr+"/test.Greeter/SayHello",
		"application/json", strings.NewReader(`{"name":"x"}`))

	if !logged {
		t.Fatal("middleware was not called")
	}
}

func TestRegisterHandler(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterHandler(srv, "test.Greeter/SayHello", (&testService{}).SayHello)

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Post(
		"http://"+addr+"/test.Greeter/SayHello",
		"application/json",
		strings.NewReader(`{"name":"World"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}

func TestRegisterUnary(t *testing.T) {
	srv := NewServer(Address(":0"))
	RegisterUnary(srv, "GET", "/api/v1/hello/:name",
		(&testService{}).SayHello,
		func(ctx Context, req *helloReq) error {
			req.Name = ctx.PathValue("name")
			return nil
		},
	)

	addr, stop := startServer(t, srv)
	defer stop()

	resp, err := http.Get("http://" + addr + "/api/v1/hello/World")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reply helloReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Message != "Hello World" {
		t.Fatalf("got %q, want %q", reply.Message, "Hello World")
	}
}
