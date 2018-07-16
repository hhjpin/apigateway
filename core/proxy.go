package core

import (
	"github.com/valyala/fasthttp"
	"time"
	"api_gateway/utils/errors"
	"bytes"
)

func MainRequestHandlerWrapper(table *RoutingTable) fasthttp.RequestHandler {
	return fasthttp.TimeoutHandler(
		func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("RoutingTable", table)
			ReverseProxyHandler(ctx)
	},
	time.Second * 5,
	errors.New(5).Error(),
)
}

func ReverseProxyHandler(ctx *fasthttp.RequestCtx) {

	path := ctx.Path()
	logger := ctx.Logger()
	routingTable := ctx.UserValue("RoutingTable")
	if routingTable == nil {
		logger.Printf("Routing Table not exists")
		ctx.Error(errors.New(7).MarshalString(), fasthttp.StatusInternalServerError)
	}

	rt, ok := routingTable.(*RoutingTable)
	if !ok {
		logger.Printf("wrong type of Routing Table")
		ctx.Error(errors.New(7).MarshalString(), fasthttp.StatusInternalServerError)
	}

	target, err := rt.Select(path)
	if err != nil {
		logger.Printf(err.Error())
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
	logger.Printf("reverse request header: %+v", revReq.Header)

	if body := ctx.Request.Body(); len(body) > 0 {
		revReq.SetBody(body)
	}

	err = fasthttp.Do(revReq, revRes)
	if err != nil {
		logger.Printf("http client raised an error: %+v", err)
	}
	ctx.SetBody(revRes.Body())
}
