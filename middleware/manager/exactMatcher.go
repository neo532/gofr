package manager

import (
	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

type ExactMatcher struct {
	paths []string
	ms    []middleware.Middleware
}

func NewExactMatcher(paths ...string) *ExactMatcher {
	return &ExactMatcher{
		paths: paths,
	}
}

func (m *ExactMatcher) Use(ms ...middleware.Middleware) Matcher {
	m.ms = ms
	return m
}

func (m *ExactMatcher) Match(op transport.Operation) (ms []middleware.Middleware) {
	for _, p := range m.paths {
		if p == op.Operation {
			return m.ms
		}
	}
	return
}
