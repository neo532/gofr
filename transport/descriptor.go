package transport

import "context"

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, request any) (response any, err error)
