package manager

import (
	"strings"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

type PrefixMatcher struct {
	prefixes []string
	ms       []middleware.Middleware
}

func NewPrefixMatcher(prefixes ...string) *PrefixMatcher {
	return &PrefixMatcher{
		prefixes: prefixes,
	}
}
func (m *PrefixMatcher) Use(ms ...middleware.Middleware) Matcher {
	m.ms = ms
	return m
}
func (m *PrefixMatcher) Match(op transport.Operation) (ms []middleware.Middleware) {
	for _, pf := range m.prefixes {
		if strings.HasPrefix(op.Operation, pf) {
			return m.ms
		}
	}
	return
}
