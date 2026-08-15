package http

import "testing"

func TestRouteRegistry(t *testing.T) {
	RegisterRoutes(map[string]Route{
		"user.api.user.UserApi/GetByUserId": {
			Service:    "user.api.user.UserApi",
			Method:     "GetByUserId",
			HTTPMethod: "GET",
			Path:       "/user/{userId}",
		},
	})

	r, ok := LookupRoute("user.api.user.UserApi/GetByUserId")
	if !ok {
		t.Fatal("route not found")
	}
	if r.HTTPMethod != "GET" || r.Path != "/user/{userId}" {
		t.Fatalf("got %+v", r)
	}
	if r.FullMethod() != "user.api.user.UserApi/GetByUserId" {
		t.Fatalf("FullMethod = %q", r.FullMethod())
	}

	if _, ok := LookupRoute("no.Such/Method"); ok {
		t.Fatal("unexpected route found")
	}
}
