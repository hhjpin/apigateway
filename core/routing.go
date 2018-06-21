package core

import (
	"api_gateway/middleware"
	"log"
	"sync"
)

type RouteTable struct {
	Version []byte

	table       RouteTableMap
	onlineTable OnlineRouteTableMap
	srvTable    ServerTableMap
}

type Api struct {
	path []byte
	pathString string
}

type HealthCheck struct {
	path      []byte
	timeout   uint8
	interval  uint8
	retry     bool
	retryTime uint8
}

type RateLimit struct {
}

type Server struct {
	host   []byte
	port   uint8
	status uint8

	healthCheck HealthCheck
	rate        RateLimit
}

type Service struct {
	name []byte
	srv  sync.Map

	// if request method not in the accept http method slice, return HTTP 405
	// if AcceptHttpMethod slice is empty, allow all http verb.
	acceptHttpMethod [][]byte
}

type Router struct {
	name   []byte
	status uint8

	frontendApi *Api
	backendApi  *Api
	svr         *Service

	middleware []*middleware.Middleware
}

func CreateApi() {}

func RemoveApi() {}

func CreateServer() {}

func RemoveServer() {}

func CreateService() {}

func RemoveService() {}

func CreateRouter() {}

func (s *Service) UpdateServer() {}

func (r *RouteTable) RegisterRouter(router *Router) error {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if exists {
		log.SetPrefix("[WARNING]")
		log.Print("router already exists")
		return nil
	}

	r.table.Store(router.frontendApi.pathString, router)

	return nil
}

func (r *RouteTable) UnregisterRouter(router *Router) error {

	return nil
}
