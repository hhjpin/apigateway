package main

import (
	"api_gateway/core"
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

		Name:               Conf.Server.Name,
		Concurrency:        Conf.Server.Concurrency,
		ReadBufferSize:     Conf.Server.ReadBufferSize,
		WriteBufferSize:    Conf.Server.WriteBufferSize,
		DisableKeepalive:   Conf.Server.DisabledKeepAlive,
		ReduceMemoryUsage:  Conf.Server.ReduceMemoryUsage,
		MaxRequestBodySize: Conf.Server.MaxRequestBodySize,
	}

	host := fmt.Sprintf("%s:%d", Conf.Server.ListenHost, Conf.Server.ListenPort)
	log.Print(host)
	err := server.ListenAndServe(host)
	if err != nil {
		log.Fatal(err)
	}
}
