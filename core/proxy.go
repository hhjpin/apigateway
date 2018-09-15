package core

import (
	"api_gateway/middleware"
	"bytes"
	"git.henghajiang.com/backend/golang_utils/errors"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/valyala/fasthttp"
	"time"
)

var (
	proxyLogger = log.New()
)

func MainRequestHandlerWrapper(table *RoutingTable, middle ...*middleware.Middleware) fasthttp.RequestHandler {
	return fasthttp.TimeoutHandler(
		func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("RoutingTable", table)
			ctx.SetUserValue("Middleware", middle)
			ReverseProxyHandler(ctx)
		},
		time.Second*5,
		errors.New(5).Error(),
	)
}

func ReverseProxyHandler(ctx *fasthttp.RequestCtx) {

	path := ctx.Path()
	routingTable := ctx.UserValue("RoutingTable")
	if routingTable == nil {
		proxyLogger.Error("Routing Table not exists")
		ctx.Error(errors.New(7).MarshalString(), fasthttp.StatusInternalServerError)
	}

	rt, ok := routingTable.(*RoutingTable)
	if !ok {
		proxyLogger.Error("wrong type of Routing Table")
		ctx.Error(errors.New(7).MarshalString(), fasthttp.StatusInternalServerError)
	}

	target, err := rt.Select(path)
	if err != nil {
		proxyLogger.Exception(err)
		ctx.Error(err.(errors.Error).MarshalString(), fasthttp.StatusInternalServerError)
	}

	revReq := fasthttp.AcquireRequest()
	revReqUri := fasthttp.AcquireURI()
	revRes := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(revReq)
	defer fasthttp.ReleaseResponse(revRes)
	defer fasthttp.ReleaseURI(revReqUri)

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		if bytes.Equal(key, strHost) {
			// pass
		} else {
			revReq.Header.AppendBytes(value)
		}
	})

	revReqUri.SetHostBytes(target.host)
	revReqUri.SetPathBytes(target.uri)
	revReqUri.SetScheme("http")

	if queryString := ctx.QueryArgs().QueryString(); len(queryString) > 0 {
		revReqUri.SetQueryStringBytes(queryString)
	}
	revReq.SetRequestURIBytes(revReqUri.FullURI())
	proxyLogger.Infof("reverse request header: %+v", revReq.Header)

	if body := ctx.Request.Body(); len(body) > 0 {
		revReq.SetBody(body)
	}

	err = fasthttp.Do(revReq, revRes)
	if err != nil {
		proxyLogger.Exception(err)
	}
	ctx.SetBody(revRes.Body())
}
