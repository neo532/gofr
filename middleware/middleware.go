package middleware

import "github.com/neo532/gofr/transport"

// Middleware wraps a Handler to add cross-cutting behavior.
type Middleware func(transport.Handler) transport.Handler

// Chain composes middlewares into a single one.
// The first middleware becomes the outermost layer.
func Chain(m ...Middleware) Middleware {
	return func(next transport.Handler) transport.Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}
