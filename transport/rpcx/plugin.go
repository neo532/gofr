package rpcx

import (
	"context"

	"github.com/smallnest/rpcx/share"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// middlewarePlugin adapts MiddlewareManager to rpcx's PreCallPlugin/PostCallPlugin.
type middlewarePlugin struct {
	mwManager *MiddlewareManager
}

func (p *middlewarePlugin) PreCall(ctx context.Context, servicePath, serviceMethod string, args any) (any, error) {
	fullMethod := "/" + servicePath + "/" + serviceMethod

	// Inject Transporter into the mutable share.Context so that
	// transport.FromServerContext(ctx) works in both middleware and handler.
	if shareCtx, ok := ctx.(*share.Context); ok {
		reqMeta, _ := shareCtx.Value(share.ReqMetaDataKey).(map[string]string)
		if reqMeta == nil {
			reqMeta = make(map[string]string)
			share.WithLocalValue(shareCtx, share.ReqMetaDataKey, reqMeta)
		}

		resMeta, _ := shareCtx.Value(share.ResMetaDataKey).(map[string]string)
		if resMeta == nil {
			resMeta = make(map[string]string)
			share.WithLocalValue(shareCtx, share.ResMetaDataKey, resMeta)
		}

		tr := &Transport{
			operation:   fullMethod,
			reqHeader:   headerCarrier(reqMeta),
			replyHeader: headerCarrier(resMeta),
		}

		// Use the same exported key as transport.FromServerContext.
		share.WithLocalValue(shareCtx, transport.ServerTransportKey{}, tr)
	}

	// Existing middleware chain — ctx now carries the Transporter.
	matched := p.mwManager.Match(fullMethod)
	if len(matched) == 0 {
		return args, nil
	}

	chain := middleware.Chain(matched...)
	h := chain(func(ctx context.Context, req any) (any, error) {
		return req, nil
	})
	return h(ctx, args)
}

func (p *middlewarePlugin) PostCall(ctx context.Context, servicePath, serviceMethod string, args, reply any, err error) (any, error) {
	return reply, err
}
