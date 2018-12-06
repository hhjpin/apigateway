package middleware

import (
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/valyala/fasthttp"
	"os"
	"strconv"
)

type Middleware interface {
	Work(ctx *fasthttp.RequestCtx, errChan chan error)
}

var (
	logger = log.New()
)

func init() {
	serverDebug := os.Getenv("SERVER_DEBUG")
	if serverDebug != "" {
		tmp, err := strconv.ParseInt(serverDebug, 10, 64)
		if err != nil {
			logger.Exception(err)
		}
		if tmp > 0 {
			logger.EnableDebug()
		} else {
			logger.DisableDebug()
		}
	} else {
		logger.EnableDebug()
	}
}
