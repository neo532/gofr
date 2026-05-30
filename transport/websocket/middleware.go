package websocket

import "github.com/neo532/gofr/middleware"

// MiddlewareManager manages global and per-method middleware for WebSocket.
type MiddlewareManager struct {
	global []middleware.Middleware
	opMap  map[string][]middleware.Middleware
}

func newMiddlewareManager() *MiddlewareManager {
	return &MiddlewareManager{
		opMap: make(map[string][]middleware.Middleware),
	}
}

// Use registers global middlewares applied to all endpoints.
func (m *MiddlewareManager) Use(mw ...middleware.Middleware) {
	m.global = append(m.global, mw...)
}

// UseWith registers middlewares scoped to a specific endpoint path.
func (m *MiddlewareManager) UseWith(path string, mw ...middleware.Middleware) {
	m.opMap[path] = append(m.opMap[path], mw...)
}

// Match returns the combined middleware list for the given path: global + scoped.
func (m *MiddlewareManager) Match(path string) []middleware.Middleware {
	scoped := m.opMap[path]
	if len(scoped) == 0 {
		return m.global
	}
	matched := make([]middleware.Middleware, 0, len(m.global)+len(scoped))
	matched = append(matched, m.global...)
	matched = append(matched, scoped...)
	return matched
}
