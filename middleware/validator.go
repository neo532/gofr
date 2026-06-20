package middleware

import (
	"context"

	"github.com/neo532/gofr/transport"
)

// Validator returns a middleware that calls Validate() on the request
// if it implements the Validate() error interface.
func Validator() Middleware {
	return func(next transport.Handler) transport.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if v, ok := req.(interface{ Validate() error }); ok {
				if err := v.Validate(); err != nil {
					return nil, err
				}
			}
			return next(ctx, req)
		}
	}
}
