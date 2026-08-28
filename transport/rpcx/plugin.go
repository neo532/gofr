package rpcx

import (
	"context"
	"net"
	"strings"

	rpcxServer "github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"

	"github.com/neo532/gofr/middleware"
	"github.com/neo532/gofr/middleware/manager"
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gofr/transport/route"
)

// middlewarePlugin adapts MiddlewareManager to rpcx's PreCallPlugin/PostCallPlugin.
type middlewarePlugin struct {
	mwManager      *manager.MiddlewareManager
	trustedProxies []*net.IPNet
	srv            *Server
}

func (p *middlewarePlugin) PreCall(ctx context.Context, servicePath, serviceMethod string, args any) (any, error) {
	fullMethod := "/" + servicePath + "/" + serviceMethod

	op, tmpl := func() (transport.Operation, string) {
		if op, tmpl, ok := route.RouteOperation(fullMethod); ok {
			return op, tmpl
		}
		return transport.Operation{Operation: fullMethod}, ""
	}()

	// Inject Transporter into the mutable share.Context so that
	// transport.FromServerContext(ctx) works in both middleware and handler.
	var shareCtx *share.Context
	var tr *Transport
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

		tr = &Transport{
			operation:      op.Operation,
			method:         op.Method,
			routeKey:       strings.TrimPrefix(fullMethod, "/"),
			pathTmpl:       tmpl,
			reqHeader:      headerCarrier(reqMeta),
			replyHeader:    headerCarrier(resMeta),
			trustedProxies: p.trustedProxies,
			// Read the App at request time: App() is called by gofr.App.Run
			// after NewServer, so capturing it at construction would freeze nil.
			app: p.srv.app,
		}
		tr.SetReq(args)
		if conn, ok := shareCtx.Value(rpcxServer.RemoteConnContextKey).(net.Conn); ok && conn.RemoteAddr() != nil {
			tr.peer = conn.RemoteAddr().String()
		}

		// Use the same exported key as transport.FromServerContext.
		share.WithLocalValue(shareCtx, transport.ServerTransportKey{}, tr)
	}

	// Existing middleware chain — ctx now carries the Transporter. rpcx always
	// passes a *share.Context, so tr is set; the nil check is purely defensive.
	if tr == nil {
		return args, nil
	}
	matched := p.mwManager.Match(tr.Operation())
	if len(matched) == 0 {
		return args, nil
	}

	chain := middleware.Chain(matched...)
	var chainCtx context.Context
	h := chain(func(ctx context.Context, req any) (any, error) {
		chainCtx = ctx
		return req, nil
	})

	// rpcx calls the service with the same share.Context, so values installed by
	// middleware (e.g. PackageContextArguments' request args) must be published
	// onto it to reach the handler. Run the chain on a base detached from
	// shareCtx: chainCtx would otherwise derive from shareCtx, and assigning that
	// as shareCtx.Context would recurse through shareCtx.Value. The Transporter
	// (normally carried in the tags) is put on the base explicitly, then the
	// final chain context replaces shareCtx's base so the handler resolves every
	// middleware value, including the trace span.
	base := ctx
	if shareCtx != nil {
		base = context.WithValue(shareCtx.Context, transport.ServerTransportKey{}, tr)
	}
	out, err := h(base, args)
	if err != nil {
		return out, err
	}
	if shareCtx != nil && chainCtx != nil {
		shareCtx.Context = chainCtx
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
