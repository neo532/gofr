package http

import (
	"context"

	"github.com/neo532/gofr/middleware"
)

// RegisterHandler is a generic registration for a single method with zero reflection at request time.
func RegisterHandler[Req, Res any](
	s *Server,
	operation string,
	fn func(context.Context, *Req) (*Res, error),
) {
	path := "/" + operation

	matched := s.mwManager.Match(path)
	wrapped := func(ctx context.Context, req any) (any, error) {
		return fn(ctx, req.(*Req))
	}
	prebuilt := middleware.Chain(matched...)(wrapped)

	s.Handle("POST", path, func(ctx Context) error {
		var req Req

		if err := ctx.Bind(&req); err != nil {
			return err
		}

		out, err := prebuilt(ctx, &req)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}

// RegisterUnary is a generic route registration function called by generated code.
// Middleware chain is pre-built at registration, zero allocation and zero reflection at request time.
func RegisterUnary[Req, Res any](
	s *Server,
	method, path string,
	fn func(context.Context, *Req) (*Res, error),
	dec func(Context, *Req) error,
) {
	wrapped := func(ctx context.Context, req any) (any, error) {
		return fn(ctx, req.(*Req))
	}
	matched := s.mwManager.Match(path)
	prebuilt := middleware.Chain(matched...)(wrapped)

	s.Handle(method, path, func(ctx Context) error {
		var req Req
		if err := dec(ctx, &req); err != nil {
			return err
		}
		out, err := prebuilt(ctx, &req)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}
