package core

import (
	"api_gateway/middleware"
	"bytes"
	"fmt"
	"git.henghajiang.com/backend/golang_utils/errors"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/go-ego/murmur"
	"github.com/valyala/fasthttp"
	"strconv"
	"time"
)

var (
	proxyLogger = log.New()
)

func MainRequestHandlerWrapper(table *RoutingTable, middle ...middleware.Middleware) fasthttp.RequestHandler {
	return fasthttp.TimeoutHandler(
		func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("RoutingTable", table)
			if len(middle) > 0 {
				errChan := make(chan error, len(middle))
				for _, m := range middle {
					go m.Work(ctx, errChan)
				}
				timer := time.NewTimer(1 * time.Second)
				for i := 0; i < len(middle); i++ {
					timer.Reset(1 * time.Second)
					select {
					case <-timer.C:
						ctx.Response.SetStatusCode(fasthttp.StatusOK)
						ctx.Response.Header.Set("Server", "Api Gateway")
						ctx.Response.Header.SetContentTypeBytes(strApplicationJson)
						ctx.Response.SetBody(errors.New(5).MarshalEmptyData())
						return
					case e := <-errChan:
						if e != nil {
							ctx.Response.SetStatusCode(fasthttp.StatusOK)
							ctx.Response.Header.Set("Server", "Api Gateway")
							ctx.Response.Header.SetContentTypeBytes(strApplicationJson)
							if err, ok := e.(errors.Error); ok {
								ctx.Response.SetBody(err.MarshalEmptyData())
								return
							} else {
								ctx.Response.SetBody(errors.New(1).MarshalEmptyData())
								return
							}
						}
					}
				}
			}
			ReverseProxyHandler(ctx)
			return
		},
		time.Second*5,
		errors.New(5).Error(),
	)
}

func ReverseProxyHandler(ctx *fasthttp.RequestCtx) {
	var target TargetServer

	path := ctx.Path()
	routingTable := ctx.UserValue("RoutingTable")

	revReq := fasthttp.AcquireRequest()
	revReqUri := fasthttp.AcquireURI()
	revRes := fasthttp.AcquireResponse()
	revReqHeader := make(map[string]string)
	revReqUrlParam := make(map[string]string)

	defer fasthttp.ReleaseRequest(revReq)
	defer fasthttp.ReleaseResponse(revRes)
	defer fasthttp.ReleaseURI(revReqUri)

	if routingTable == nil {
		proxyLogger.Error("Routing Table not exists")
		ctx.Error(string(errors.New(7).MarshalEmptyData()), fasthttp.StatusInternalServerError)
		return
	}

	rt, ok := routingTable.(*RoutingTable)
	if !ok {
		proxyLogger.Error("wrong type of Routing Table")
		ctx.Error(string(errors.New(7).MarshalEmptyData()), fasthttp.StatusInternalServerError)
		return
	}

	target, err := rt.Select(path)
	if err != nil {
		proxyLogger.Exception(err)
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
		if bytes.Equal(key, strHost) {
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
	fmt.Println(string(revReq.Header.Method()))
	err = fasthttp.Do(revReq, revRes)
	if err != nil {
		proxyLogger.Exception(err)
		ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
		return
	}
	revRes.Header.VisitAll(func(key, value []byte) {
		if bytes.Equal(key, strHost) {
			// pass
		} else {
			ctx.Response.Header.AppendBytes(value)
		}
	})
	ctx.Response.SetStatusCode(revRes.StatusCode())
	ctx.Response.Header.SetContentTypeBytes(revRes.Header.ContentType())
	ctx.SetBody(revRes.Body())

	defer func() {
		var resContentType []byte

		if revRes != nil {
			resContentType = revRes.Header.ContentType()
		}

		counting := middleware.NewCounting(
			ctx.ConnTime().UnixNano(),
			time.Now().UnixNano(),
			ctx.Path(),
			ctx.Method(),
			target.svr,
			target.host,
			target.uri,
			ctx.Request.Header.ContentType(),
			resContentType,
			revReqHeader,
			revReqUrlParam,
			ctx.Response.StatusCode(),
			ctx.Request.Body(),
			ctx.Response.Body(),
		)
		go func() {
			hashed := murmur.Sum32(strconv.FormatUint(ctx.ConnID(), 10)) % middleware.CountingShardNumber
			timer := time.NewTimer(5 * time.Second)
			select {
			case <-timer.C:
				proxyLogger.Warning("Counting channel maybe full")
			case middleware.CountingCh[hashed] <- counting:
				// pass
			}
		}()
	}()
}
