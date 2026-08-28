package manager

import (
	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

type Matcher interface {
	Use(ms ...middleware.Middleware) Matcher
	Match(op transport.Operation) []middleware.Middleware
}

type MiddlewareManager struct {
	global []middleware.Middleware
	ms     []Matcher
}

func NewMiddlewareManager(ms ...Matcher) (m *MiddlewareManager) {
	m = &MiddlewareManager{
		ms:     ms,
		global: make([]middleware.Middleware, 0, 1),
	}
	return
}

// Use adds global middleware applied to every path.
func (m *MiddlewareManager) Use(mw ...middleware.Middleware) *MiddlewareManager {
	m.global = append(m.global, mw...)
	return m
}

// Global returns a copy of the currently registered global middlewares.
// A server uses it to carry existing globals over when swapping in an
// external manager, so option order does not matter.
func (m *MiddlewareManager) Global() (ms []middleware.Middleware) {
	ms = append(ms, m.global...)
	return
}

// Match returns the middleware chain for an operation: global, then the matchers
// (matched against the operation's proto route template), or nil when none apply.
func (m *MiddlewareManager) Match(op transport.Operation) (ms []middleware.Middleware) {

	ms = make([]middleware.Middleware, 0, 10)
	ms = append(ms, m.global...)

	for _, v := range m.ms {
		if vs := v.Match(op); len(vs) > 0 {
			ms = append(ms, vs...)
		}
	}
	return
}
