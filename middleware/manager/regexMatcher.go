package manager

import (
	"regexp"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

type RegexMatcher struct {
	exp *regexp.Regexp
	ms  []middleware.Middleware
}

func NewRegexMatcher(exp *regexp.Regexp) *RegexMatcher {
	return &RegexMatcher{
		exp: exp,
	}
}

func (m *RegexMatcher) Use(ms ...middleware.Middleware) Matcher {
	m.ms = ms
	return m
}

func (m *RegexMatcher) Match(op transport.Operation) (ms []middleware.Middleware) {
	if m.exp.MatchString(op.Operation) {
		return m.ms
	}
	return
}
