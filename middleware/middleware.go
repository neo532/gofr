package middleware

import "context"

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, request any) (response any, err error)

// Middleware wraps a Handler to add cross-cutting behavior.
type Middleware func(Handler) Handler

// Chain composes middlewares into a single one.
// The first middleware becomes the outermost layer.
func Chain(m ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}
