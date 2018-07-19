package main

import (
	"api_gateway/core"
	"api_gateway/utils"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
)

var (
	table *core.RoutingTable
)

func init() {
	table = core.InitRoutingTable()
}

func main() {
	var server *fasthttp.Server

	server = &fasthttp.Server{
		Handler: core.MainRequestHandlerWrapper(table),

		Name:               utils.Conf.Server.Name,
		Concurrency:        utils.Conf.Server.Concurrency,
		ReadBufferSize:     utils.Conf.Server.ReadBufferSize,
		WriteBufferSize:    utils.Conf.Server.WriteBufferSize,
		DisableKeepalive:   utils.Conf.Server.DisabledKeepAlive,
		ReduceMemoryUsage:  utils.Conf.Server.ReduceMemoryUsage,
		MaxRequestBodySize: utils.Conf.Server.MaxRequestBodySize,
	}

	host := fmt.Sprintf("%s:%d", utils.Conf.Server.ListenHost, utils.Conf.Server.ListenPort)
	log.Print(host)
	err := server.ListenAndServe(host)
	if err != nil {
		log.Fatal(err)
	}
}
