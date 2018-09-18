package middleware

import "github.com/valyala/fasthttp"

type Middleware interface {
	Work(ctx *fasthttp.RequestCtx) error
}
