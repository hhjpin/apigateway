package main

import (
	"api_gateway/utils"
	"fmt"
	"github.com/valyala/fasthttp"
	"api_gateway/core"
)

var (

)

func init() {

}

func main() {
	var server *fasthttp.Server

	server = &fasthttp.Server{
		Handler: core.MainRequestHandler,

		Name:               utils.Conf.Server.Name,
		Concurrency:        utils.Conf.Server.Concurrency,
		ReadBufferSize:     utils.Conf.Server.ReadBufferSize,
		WriteBufferSize:    utils.Conf.Server.WriteBufferSize,
		DisableKeepalive:   utils.Conf.Server.DisabledKeepAlive,
		ReduceMemoryUsage:  utils.Conf.Server.ReduceMemoryUsage,
		MaxRequestBodySize: utils.Conf.Server.MaxRequestBodySize,
	}

	server.ListenAndServe(fmt.Sprintf("%s:%d", utils.Conf.Server.ListenHost, utils.Conf.Server.ListenPort))
}
