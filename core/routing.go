package core

import (
	"api_gateway/middleware"
	"log"
)

type RouteTable struct {
	Version []byte

	// frontend-api/router mapping table
	table       RouteTableMap
	// online frontend-api/router mapping table
	onlineTable OnlineRouteTableMap
	// service/router reverse mapping table
	srvTable    ServiceTableMap
}

type Api struct {
	path       []byte
	// string type of path variable, used for indexing mapping table
	pathString string
}

type HealthCheck struct {
	path      []byte

	// timeout default is 5 sec. When timed out, will retry to check again based on 'retry' switch on or not
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
	status uint8 // 0 -> offline, 1 -> online, 2 -> breakdown

	healthCheck HealthCheck
	rate        RateLimit
}

type Service struct {
	name []byte
	srv  ServerTableMap

	// if request method not in the accept http method slice, return HTTP 405
	// if AcceptHttpMethod slice is empty, allow all http verb.
	acceptHttpMethod [][]byte
}

type Router struct {
	name   []byte
	status uint8 // 0 -> offline, 1 -> online, 2 -> breakdown

	frontendApi *Api
	backendApi  *Api
	svr         *Service

	middleware []*middleware.Middleware
}

func (r *RouteTable) CreateFrontendApi(path []byte) (api *Api, ok bool) {
	if router, exists := r.table.Load(string(path)); exists {
		ok = true
		api = router.frontendApi
	} else {
		ok = false
		api = &Api{path: path, pathString: string(path)}
	}
	return api, ok
}

func (r *RouteTable) RemoveFrontendApi(path []byte) (ok bool) {
	if router, exists := r.table.Load(string(path)); exists {
		if router.status == 1 {
			// router is online, api can not be deleted
			return false
		} else {
			// remove frontend api from router obj.
			router.frontendApi = nil
			return true
		}
	} else {
		return false
	}
}

func (r *RouteTable) CreateServer(host []byte, port uint8) {}

func RemoveServer() {}

func CreateService() {}

func RemoveService() {}

func CreateRouter() {}

func (s *Service) UpdateServer(host []byte, port uint8) {}

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
