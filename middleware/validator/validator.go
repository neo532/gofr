package validator

import (
	"context"

	"github.com/neo532/gofr/middleware"
)

// Validator returns a middleware that calls Validate() on the request
// if it implements the Validate() error interface.
func Validator() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if v, ok := req.(interface{ Validate() error }); ok {
				if err := v.Validate(); err != nil {
					return nil, err
				}
			}
			return next(ctx, req)
		}
	}
}
