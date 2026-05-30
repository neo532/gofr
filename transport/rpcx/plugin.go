package rpcx

import (
	"context"

	"github.com/neo532/gofr/middleware"
)

// middlewarePlugin adapts MiddlewareManager to rpcx's PreCallPlugin/PostCallPlugin.
type middlewarePlugin struct {
	mwManager *MiddlewareManager
}

func (p *middlewarePlugin) PreCall(ctx context.Context, servicePath, serviceMethod string, args interface{}) (interface{}, error) {
	fullMethod := "/" + servicePath + "/" + serviceMethod
	matched := p.mwManager.Match(fullMethod)
	if len(matched) == 0 {
		return args, nil
	}

	// Build middleware chain where the inner handler passes args through
	chain := middleware.Chain(matched...)
	h := chain(func(ctx context.Context, req interface{}) (interface{}, error) {
		return req, nil
	})
	return h(ctx, args)
}

func (p *middlewarePlugin) PostCall(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}, err error) (interface{}, error) {
	return reply, err
}
