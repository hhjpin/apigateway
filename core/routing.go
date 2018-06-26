/*
	Defined all component made up the routing table.
	Component:
		Api
		Router
		Service
		Server
	Relation between component:
		Router:
			Front Api
			Backend Api
			Service
		Service:
			[]Server
*/
package core

import (
	"api_gateway/middleware"
	"log"
	"github.com/kataras/iris/core/errors"
)

const (
	Offline   Status = iota
	Online
	BreakDown
)

type Status uint8
type FrontApiString string
type ServerNameString string
type ServiceNameString string

// Routing table struct, should not be copied at any time. Using function `Init()` to create it
type RouteTable struct {
	Version string

	// frontend-api/router mapping table
	table RouteTableMap
	// online frontend-api/router mapping table
	onlineTable OnlineRouteTableMap
	// service table
	serviceTable ServiceTableMap
	// server table
	serverTable ServerTableMap
}

// Api-struct used to create Frontend-Api or Backend-Api. <string> type attr `pathString` is using on mapping tables
type Api struct {
	path []byte
	// string type of path variable, used for indexing mapping table
	pathString FrontApiString
}

// Server-struct defined a backend-server
type Server struct {
	name []byte
	nameString ServerNameString

	host   []byte
	port   uint8
	status Status // 0 -> offline, 1 -> online, 2 -> breakdown

	healthCheck *HealthCheck
	rate        *RateLimit
}

// Service-struct defined a backend-service
type Service struct {
	name []byte
	nameString ServiceNameString
	srv  ServerTableMap

	// if request method not in the accept http method slice, return HTTP 405
	// if AcceptHttpMethod slice is empty, allow all http verb.
	acceptHttpMethod [][]byte
}

type Router struct {
	name   []byte
	status Status // 0 -> offline, 1 -> online, 2 -> breakdown

	frontendApi *Api
	backendApi  *Api
	svr         *Service

	middleware []*middleware.Middleware
}

// create frontend api obj, return pointer of api obj. if already exists, return that one
func (r *RouteTable) CreateFrontendApi(path []byte) (api *Api, ok bool) {

	if router, exists := r.table.Load(FrontApiString(path)); exists {
		ok = true
		api = router.frontendApi
	} else {
		ok = false
		api = &Api{path: path, pathString: FrontApiString(path)}
	}
	return api, ok
}

func (r *RouteTable) RemoveFrontendApi(path []byte) (ok bool, err error) {
	if router, exists := r.table.Load(FrontApiString(path)); exists {
		if router.status == Online {
			// router is online, api can not be deleted
			return false, errors.New("router is online, can not remove frontend api")
		} else {
			// remove frontend api from router obj.
			router.frontendApi = nil
			return true, nil
		}
	} else {
		return false, errors.New("router not exist")
	}
}

func (r *RouteTable) RegisterRouter(router *Router) (ok bool, err error) {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if exists {
		log.SetPrefix("[WARNING]")
		log.Print("router already exists")
		return false, errors.New("router already exists")
	}

	// newly register can not be online
	if router.status == Online {
		router.status = Offline
	}

	r.table.Store(router.frontendApi.pathString, router)

	return true, nil
}

func (r *RouteTable) UnregisterRouter(router *Router) (ok bool, err error) {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		log.SetPrefix("[WARNING]")
		log.Print("router not exists")
		return false, errors.New("router not exist")
	}

	if router.status == Online {
		// router is online, router can not be unregistered
		return false, errors.New("router is online, can not unregister it")
	}

	r.table.Delete(router.frontendApi.pathString)

	return true, nil
}

func (r *RouteTable) addBackendService(service *Service) (ok bool, err error) {

	_, exists := r.serviceTable.Load(service.nameString)
	if exists {
		log.SetPrefix("[INFO]")
		log.Print("service already exists")
		return false, errors.New("service already exists")
	}
	r.serviceTable.Store(service.nameString, service)
	return true, nil
}

func (r *RouteTable) removeBackendService(service *Service) (ok bool, err error) {

	_, exists := r.serviceTable.Load(service.nameString)
	if !exists {
		log.SetPrefix("[INFO]")
		log.Print("service not exist")
		return false, errors.New("service not exists")
	}
	r.serviceTable.Delete(service.nameString)
	return true, nil
}

func (r *RouteTable) addBackendServer(server *Server) (ok bool, err error) {

	_, exists := r.serverTable.Load(server.nameString)
	if exists {
		log.SetPrefix("[INFO]")
		log.Print("server already exists")
		return false, errors.New("server already exists")
	}
	r.serverTable.Store(server.nameString, server)
	return true, nil
}

func (r *RouteTable) removeBackendServer(server *Server) (ok bool, err error) {

	_, exists := r.serverTable.Load(server.nameString)
	if !exists {
		log.SetPrefix("[INFO]")
		log.Print("server not exist")
		return false, errors.New("server not exists ")
	}
	r.serverTable.Delete(server.nameString)
	return true, nil
}

func (r *RouteTable) serverExists(server *Server) bool {
	_, exists := r.serverTable.Load(server.nameString)
	return exists
}

// check input server exist in the server-table or not, return the rest not-existed server slice
func (r *RouteTable) serverSliceExists(server []*Server) (rest []*Server){
	if len(server) > 0 {
		for _, s := range server {
			if !r.serverExists(s) {
				rest = append(rest, s)
			}
		}
	}
	return rest
}

func (r *RouteTable) SetRouterOnline(pathString FrontApiString) (ok bool, err error) {

	router, exists := r.table.Load(pathString)
	if !exists {
		log.SetPrefix("[WARNING]")
		log.Print("router not exists")
		return false, errors.New("router not exist")
	}

	onlineRouter, exists := r.onlineTable.Load(pathString)
	if !exists {
		// not exist in online table
		// check backend server status first
		onlineServer, _ := router.svr.checkServerStatus(Online)
		if len(onlineServer) == 0 {
			// all server are not online, this router can not be set to online
			return false, errors.New("all server are not online, this router should not be set to online")
		} else {
			rest := r.serverSliceExists(onlineServer)
			if len(rest) > 0 {
				// some member in <slice> onlineServer is not exist in r.serverTable
				// this will happen at data-error
				for _, i:= range rest {
					_, e := r.addBackendServer(i)
					if e != nil {
						log.SetPrefix("[ERROR]")
						log.Print("error raised when add server to server-table")
						return false, errors.New("error raised when add server to server-table")
					}
				}
			}
			router.SetOnline()
			r.onlineTable.Store(pathString, router)
			return true, nil
		}
	} else {
		// exist in online table
		// check router == onlineRouter
		if router != onlineRouter {
			// data mapping error
			return false, errors.New("data mapping error")
		}
		return false, errors.New("")
	}
}

func (r *RouteTable) SetRouterStatus(pathString FrontApiString, status Status) (ok bool, err error) {
	if status == Online {
		return r.SetRouterOnline(pathString)
	}

	router, exists := r.table.Load(pathString)
	if !exists {
		log.SetPrefix("[WARNING]")
		log.Print("router not exists")
		return false, errors.New("router not exist")
	}

	_, exists = r.onlineTable.Load(pathString)
	if !exists {
		// not exist in online table, do nothing
	} else {
		r.onlineTable.Delete(pathString)
	}
	router.setStatus(status)
	return true, nil
}

func (r *Router) setStatus(status Status) {
	r.status = status
}

func (r *Router) SetOnline(){
	r.setStatus(Online)
}

func (r *Router) SetOffline(){
	r.setStatus(Offline)
}

func (r *Router) BreakDown(){
	r.setStatus(BreakDown)
}

// check status of all server under the same service, must means must-condition status, return the rest server-pointer
// which not confirmed to the must-condition
func (s *Service) checkServerStatus(must Status) (confirm []*Server, rest []*Server){
	s.srv.Range(func(key ServerNameString, value *Server) {
		if value.status != must {
			rest = append(rest, value)
		} else {
			confirm = append(confirm, value)
		}
	})
	return confirm, rest
}
