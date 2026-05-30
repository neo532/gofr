package rpcx

import "github.com/neo532/gofr/middleware"

// MiddlewareManager manages middleware by rpcx method pattern.
type MiddlewareManager struct {
	global []middleware.Middleware
	opMap  map[string][]middleware.Middleware
}

func newMiddlewareManager() *MiddlewareManager {
	return &MiddlewareManager{
		opMap: make(map[string][]middleware.Middleware),
	}
}

// Use adds global middleware.
func (m *MiddlewareManager) Use(mw ...middleware.Middleware) {
	m.global = append(m.global, mw...)
}

// UseWith adds middleware scoped to a specific method path (e.g. "/helloworld.Greeter/SayHello").
func (m *MiddlewareManager) UseWith(method string, mw ...middleware.Middleware) {
	m.opMap[method] = append(m.opMap[method], mw...)
}

// Match returns all middlewares matching the method (global + specific).
func (m *MiddlewareManager) Match(method string) []middleware.Middleware {
	total := len(m.global) + len(m.opMap[method])
	if total == 0 {
		return nil
	}
	ms := make([]middleware.Middleware, 0, total)
	ms = append(ms, m.global...)
	ms = append(ms, m.opMap[method]...)
	return ms
}
