package main

import (
	"config"
	"fmt"
	"github.com/valyala/fasthttp"
	"proxy"
)

var (

)

func init() {

}

func main() {
	var server *fasthttp.Server

	server = &fasthttp.Server{
		Handler: proxy.MainRequestHandler,

		Name:               config.Conf.Server.Name,
		Concurrency:        config.Conf.Server.Concurrency,
		ReadBufferSize:     config.Conf.Server.ReadBufferSize,
		WriteBufferSize:    config.Conf.Server.WriteBufferSize,
		DisableKeepalive:   config.Conf.Server.DisabledKeepAlive,
		ReduceMemoryUsage:  config.Conf.Server.ReduceMemoryUsage,
		MaxRequestBodySize: config.Conf.Server.MaxRequestBodySize,
	}

	server.ListenAndServe(fmt.Sprintf("%s:%d", config.Conf.Server.ListenHost, config.Conf.Server.ListenPort))
}
