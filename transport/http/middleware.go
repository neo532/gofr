package http

import "github.com/neo532/gofr/middleware"

// MiddlewareManager manages middleware by operation pattern.
type MiddlewareManager struct {
	global []middleware.Middleware
	opMap  map[string][]middleware.Middleware // operation → middlewares
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

// UseWith adds middleware scoped to a specific operation.
func (m *MiddlewareManager) UseWith(operation string, mw ...middleware.Middleware) {
	m.opMap[operation] = append(m.opMap[operation], mw...)
}

// Match returns all middlewares matching the operation (global + specific).
func (m *MiddlewareManager) Match(operation string) []middleware.Middleware {
	total := len(m.global) + len(m.opMap[operation])
	if total == 0 {
		return nil
	}
	ms := make([]middleware.Middleware, 0, total)
	ms = append(ms, m.global...)
	ms = append(ms, m.opMap[operation]...)
	return ms
}
