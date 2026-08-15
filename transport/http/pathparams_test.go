package http

import "testing"

func TestPathParamsRegistry(t *testing.T) {
	RegisterPathParams(map[string]PathParamFunc{
		"user.api.user.UserApi/GetByUserId": func(m any) map[string]string {
			return map[string]string{"userId": "163"}
		},
	})

	fn, ok := LookupPathParams("user.api.user.UserApi/GetByUserId")
	if !ok {
		t.Fatal("provider not found")
	}
	if got := fn(nil)["userId"]; got != "163" {
		t.Fatalf("userId = %q, want %q", got, "163")
	}

	if _, ok := LookupPathParams("no.Such/Method"); ok {
		t.Fatal("unexpected provider found")
	}
}
