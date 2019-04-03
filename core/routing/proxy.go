package routing

import (
	"bytes"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"git.henghajiang.com/backend/golang_utils/errors"
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
	var target TargetServer

	revReq := fasthttp.AcquireRequest()
	revReqUri := fasthttp.AcquireURI()
	revRes := fasthttp.AcquireResponse()
	revReqHeader := make(map[string]string)
	revReqUrlParam := make(map[string]string)

	defer fasthttp.ReleaseRequest(revReq)
	defer fasthttp.ReleaseResponse(revRes)
	defer fasthttp.ReleaseURI(revReqUri)

	defer func() {
		//var resContentType []byte
		//
		//if revRes != nil {
		//	resContentType = revRes.Header.ContentType()
		//}
		//
		//counting := middleware.NewCounting(
		//	ctx.ConnTime().UnixNano(),
		//	time.Now().UnixNano(),
		//	ctx.Path(),
		//	ctx.Method(),
		//	target.svr,
		//	target.host,
		//	target.uri,
		//	ctx.Request.Header.ContentType(),
		//	resContentType,
		//	revReqHeader,
		//	revReqUrlParam,
		//	ctx.Response.StatusCode(),
		//	ctx.Request.Body(),
		//	ctx.Response.Body(),
		//)
		//go func() {
		//	hashed := murmur.Sum32(strconv.FormatUint(ctx.ConnID(), 10)) % middleware.CountingShardNumber
		//	timer := time.NewTimer(5 * time.Second)
		//	select {
		//	case <-timer.C:
		//		logger.Warning("Counting channel maybe full")
		//	case middleware.CountingCh[hashed] <- counting:
		//		// pass
		//	}
		//}()
	}()

	path := ctx.Path()
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

	target, err := rt.Select(path)
	if err != nil {
		logger.Exception(err)
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
			// pass
		} else {
			revReq.Header.AddBytesKV(key, value)
		}
		revReqHeader[string(key)] = string(value)
	})

	revReqUri.SetHostBytes(target.host)
	revReqUri.SetPathBytes(target.uri)
	revReqUri.SetScheme("http")

	if queryString := ctx.QueryArgs().QueryString(); len(queryString) > 0 {
		revReqUri.SetQueryStringBytes(queryString)
	}
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		revReqUrlParam[string(key)] = string(value)
	})
	revReq.SetRequestURIBytes(revReqUri.FullURI())

	if body := ctx.Request.Body(); len(body) > 0 {
		revReq.SetBody(body)
	}
	revReq.Header.SetMethodBytes(ctx.Request.Header.Method())
	err = fasthttp.Do(revReq, revRes)
	if err != nil {
		logger.Exception(err)
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
