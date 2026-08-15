package rpcx

import (
	"context"
	"net"

	rpcxServer "github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"

	gofrTrace "github.com/neo532/gofr/middleware/trace"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/transport"
)

// middlewarePlugin adapts MiddlewareManager to rpcx's PreCallPlugin/PostCallPlugin.
type middlewarePlugin struct {
	mwManager      *MiddlewareManager
	trustedProxies []*net.IPNet
	srv            *Server
}

func (p *middlewarePlugin) PreCall(ctx context.Context, servicePath, serviceMethod string, args any) (any, error) {
	fullMethod := "/" + servicePath + "/" + serviceMethod

	// Inject Transporter into the mutable share.Context so that
	// transport.FromServerContext(ctx) works in both middleware and handler.
	var shareCtx *share.Context
	if sc, ok := ctx.(*share.Context); ok {
		shareCtx = sc
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
			operation:      fullMethod,
			reqHeader:      headerCarrier(reqMeta),
			replyHeader:    headerCarrier(resMeta),
			trustedProxies: p.trustedProxies,
			// Read the App at request time: App() is called by gofr.App.Run
			// after NewServer, so capturing it at construction would freeze nil.
			app: p.srv.app,
		}
		if conn, ok := shareCtx.Value(rpcxServer.RemoteConnContextKey).(net.Conn); ok && conn.RemoteAddr() != nil {
			tr.peer = conn.RemoteAddr().String()
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
	var chainCtx context.Context
	h := chain(func(ctx context.Context, req any) (any, error) {
		chainCtx = ctx
		return req, nil
	})
	out, err := h(ctx, args)
	if err != nil {
		return out, err
	}
	// rpcx calls the service with the same share.Context, so values installed by
	// middleware (e.g. the trace span) must be carried into it explicitly.
	if shareCtx != nil && chainCtx != nil {
		shareCtx.Context = gofrTrace.CarrySpan(shareCtx.Context, chainCtx)
	}
	return out, nil
}

func (p *middlewarePlugin) PostCall(ctx context.Context, servicePath, serviceMethod string, args, reply any, err error) (any, error) {
	// Middleware runs inside PreCall, where the real reply does not exist yet —
	// rpcx writes it into the reply pointer only after PreCall returns. Deliver
	// it now so reply-observing middleware (e.g. request-replay logging) can log
	// the actual response. PreCall and PostCall share the same *share.Context,
	// so the per-request Transport is retrievable here.
	if tr, ok := transport.FromServerContext(ctx); ok {
		if t, ok := tr.(*Transport); ok {
			t.fireReply(reply, err)
		}
	}
	return reply, err
}
