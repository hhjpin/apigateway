package routing

import (
	"bytes"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/logger"
	"github.com/valyala/fasthttp"
	"time"
)

const (
	middlewareTimeoutLimit = 1
)

func MainRequestHandlerWrapper(table *Table, middle ...middleware.Middleware) fasthttp.RequestHandler {
	return fasthttp.TimeoutHandler(
		func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			ctx.SetUserValue("Table", table)
			if len(middle) > 0 {
				errChan := make(chan error, len(middle))
				for _, m := range middle {
					go m.Work(ctx, errChan)
				}
				timer := time.NewTimer(middlewareTimeoutLimit * time.Second)
				for i := 0; i < len(middle); i++ {
					timer.Reset(1 * time.Second)
					select {
					case <-timer.C:
						ctx.Response.SetStatusCode(fasthttp.StatusOK)
						ctx.Response.Header.Set("Server", "Api Gateway")
						ctx.Response.Header.SetContentTypeBytes(constant.StrApplicationJson)
						body := errors.New(5).MarshalEmptyData()
						ctx.Response.SetBody(body)
						go middleware.Logger(ctx.Response.StatusCode(), string(ctx.Request.URI().Path()), string(ctx.Request.Header.Method()), ctx.RemoteIP().String(), start)
						return
					case e := <-errChan:
						if e != nil {
							ctx.Response.SetStatusCode(fasthttp.StatusOK)
							ctx.Response.Header.Set("Server", "Api Gateway")
							ctx.Response.Header.SetContentTypeBytes(constant.StrApplicationJson)
							if err, ok := e.(errors.Error); ok {
								ctx.Response.SetBody(err.MarshalEmptyData())
								go middleware.Logger(ctx.Response.StatusCode(), string(ctx.Request.URI().Path()), string(ctx.Request.Header.Method()), ctx.RemoteIP().String(), start)
								return
							} else {
								ctx.Response.SetBody(errors.New(1).MarshalEmptyData())
								go middleware.Logger(ctx.Response.StatusCode(), string(ctx.Request.URI().Path()), string(ctx.Request.Header.Method()), ctx.RemoteIP().String(), start)
								return
							}
						}
					}
				}
			}
			ReverseProxyHandler(ctx)
			go middleware.Logger(ctx.Response.StatusCode(), string(ctx.Request.URI().Path()), string(ctx.Request.Header.Method()), ctx.RemoteIP().String(), start)
			return
		},
		time.Second*60,
		string(errors.New(5).MarshalEmptyData()),
	)
}

func ReverseProxyHandler(ctx *fasthttp.RequestCtx) {
	revReq := fasthttp.AcquireRequest()
	revReqUri := fasthttp.AcquireURI()
	revRes := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(revReq)
	defer fasthttp.ReleaseResponse(revRes)
	defer fasthttp.ReleaseURI(revReqUri)

	routingTable := ctx.UserValue("Table")

	if routingTable == nil {
		logger.Error("Routing Table not exists")
		ctx.Error(string(errors.New(7).MarshalEmptyData()), fasthttp.StatusInternalServerError)
		return
	}

	rt, ok := routingTable.(*Table)
	if !ok {
		logger.Error("wrong type of Routing Table")
		ctx.Error(string(errors.New(7).MarshalEmptyData()), fasthttp.StatusInternalServerError)
		return
	}

	target, err := rt.Select(ctx.Path(), ctx.Method())
	if err != nil {
		logger.Error(err)
		if e, ok := err.(errors.Error); ok {
			if e.ErrCode == 142 {
				ctx.Error(string(e.MarshalEmptyData()), fasthttp.StatusNotFound)
			} else {
				ctx.Error(string(e.MarshalEmptyData()), fasthttp.StatusInternalServerError)
			}
		} else {
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		}
		return
	}

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		if bytes.Equal(key, constant.StrHost) {
			revReq.Header.AddBytesV("X-Forwarded-Host", value)
		} else if bytes.Equal(key, constant.StrContentType) {
			revReq.Header.SetContentTypeBytes(value)
		} else if bytes.Equal(key, constant.StrUserAgent) {
			revReq.Header.SetUserAgentBytes(value)
		} else {
			revReq.Header.AddBytesKV(key, value)
		}
	})

	revReqUri.SetHostBytes(target.host)
	revReqUri.SetPathBytes(target.uri)
	revReqUri.SetScheme("http")

	if queryString := ctx.QueryArgs().QueryString(); len(queryString) > 0 {
		revReqUri.SetQueryStringBytes(queryString)
	}
	revReq.SetRequestURIBytes(revReqUri.FullURI())

	if body := ctx.Request.Body(); len(body) > 0 {
		revReq.SetBody(body)
	}
	revReq.Header.SetMethodBytes(ctx.Request.Header.Method())
	err = fasthttp.Do(revReq, revRes)
	if err != nil {
		logger.Error(err)
		ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
		return
	}
	revRes.Header.VisitAll(func(key, value []byte) {
		if bytes.Equal(key, constant.StrHost) {
			// pass
		} else {
			ctx.Response.Header.SetBytesKV(key, value)
		}
	})
	ctx.Response.SetConnectionClose()
	ctx.Response.SetStatusCode(revRes.StatusCode())
	ctx.Response.Header.SetContentTypeBytes(revRes.Header.ContentType())
	ctx.SetBody(revRes.Body())
}
