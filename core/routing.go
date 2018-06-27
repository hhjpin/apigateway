/*
	Defined all component made up the routing table.
	Component:
		FrontendApi
		Router
		Service
		Server
	Relation between component:
		Router:
			Front FrontendApi
			Backend FrontendApi
			Service
		Service:
			[]Server
*/
package core

import (
	"api_gateway/middleware"
	"bytes"
	"github.com/deckarep/golang-set"
	"github.com/kataras/iris/core/errors"
	"log"
)

const (
	Offline Status = iota
	Online
	BreakDown
)

type Status uint8
type FrontendApiString string
type BackendApiString string
type ServerNameString string
type ServiceNameString string
type RouterNameString string

// Routing table struct, should not be copied at any time. Using function `Init()` to create it
type RouteTable struct {
	Version string

	// frontend-api/router mapping table
	table ApiRouterTableMap
	// online frontend-api/router mapping table
	onlineTable OnlineApiRouterTableMap
	// service table
	serviceTable ServiceTableMap
	// server table
	serverTable ServerTableMap
	// router table
	routerTable RouterTableMap
}

// FrontendApi-struct used to create Frontend-FrontendApi or Backend-FrontendApi. <string> type attr `pathString` is using on mapping tables
type FrontendApi struct {
	path []byte
	// string type of path variable, used for indexing mapping table
	pathString FrontendApiString
}

type BackendApi struct {
	path       []byte
	pathString BackendApiString
}

// Server-struct defined a backend-server
type Server struct {
	name       []byte
	nameString ServerNameString

	host   []byte
	port   uint8
	status Status // 0 -> offline, 1 -> online, 2 -> breakdown

	healthCheck *HealthCheck
	rate        *RateLimit
}

// Service-struct defined a backend-service
type Service struct {
	name       []byte
	nameString ServiceNameString
	srv        *ServerTableMap

	// if request method not in the accept http method slice, return HTTP 405
	// if AcceptHttpMethod slice is empty, allow all http verb.
	acceptHttpMethod [][]byte
}

type Router struct {
	name   []byte
	status Status // 0 -> offline, 1 -> online, 2 -> breakdown

	frontendApi *FrontendApi
	backendApi  *BackendApi
	svr         *Service

	middleware []*middleware.Middleware
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
func (r *RouteTable) serverSliceExists(server []*Server) (rest []*Server) {
	if len(server) > 0 {
		for _, s := range server {
			if !r.serverExists(s) {
				rest = append(rest, s)
			}
		}
	}
	return rest
}

func (r *RouteTable) CreateBackendApi(path []byte) *BackendApi {
	return &BackendApi{
		path:       path,
		pathString: BackendApiString(path),
	}
}

// create frontend api obj, return pointer of api obj. if already exists, return that one
func (r *RouteTable) CreateFrontendApi(path []byte) (api *FrontendApi, ok bool) {

	if router, exists := r.table.Load(FrontendApiString(path)); exists {
		ok = true
		api = router.frontendApi
	} else {
		ok = false
		api = &FrontendApi{path: path, pathString: FrontendApiString(path)}
	}
	return api, ok
}

func (r *RouteTable) RemoveFrontendApi(path []byte) (ok bool, err error) {
	if router, exists := r.table.Load(FrontendApiString(path)); exists {
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

func (r *RouteTable) CreateRouter(name []byte, fApi *FrontendApi, bApi *BackendApi, svr *Service, mw ...*middleware.Middleware) error {
	if fApi.pathString == "" || bApi.pathString == "" {
		log.SetPrefix("[ERROR]")
		log.Print("api obj not completed")
		return errors.New("api obj not completed")
	}
	if svr.nameString == "" {
		log.SetPrefix("[ERROR]")
		log.Print("service obj not completed")
		return errors.New("service obj not completed")
	}

	_, exists := r.table.Load(fApi.pathString)

	if exists {
		return errors.New("router already exists")
	} else {
		router := &Router{
			name:        name,
			status:      Offline,
			frontendApi: fApi,
			backendApi:  bApi,
			svr:         svr,
			middleware:  mw,
		}
		service, exists := r.serviceTable.Load(svr.nameString)
		if !exists {
			r.serviceTable.Store(svr.nameString, svr)
		} else {

		}
		r.table.Store(fApi.pathString, router)
		r.routerTable.Store(RouterNameString(router.name), router)
		return nil
	}
}

func (r *RouteTable) GetRouterByName(name []byte) (*Router, error) {

}

func (r *RouteTable) RemoveRouter(router *Router) (ok bool, err error) {

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

func (r *RouteTable) SetRouterOnline(pathString FrontendApiString) (ok bool, err error) {

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
				for _, i := range rest {
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

func (r *RouteTable) SetRouterStatus(pathString FrontendApiString, status Status) (ok bool, err error) {
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

func (r *RouteTable) CreateService(name []byte, acceptHttpMethod [][]byte) *Service {

	svr, exists := r.serviceTable.Load(ServiceNameString(name))
	if exists {
		return svr
	} else {
		service := &Service{
			name:             name,
			nameString:       ServiceNameString(name),
			srv:              &ServerTableMap{},
			acceptHttpMethod: acceptHttpMethod,
		}
		r.serviceTable.Store(service.nameString, service)
		return service
	}
}

func (r *RouteTable) CreateServer(name, host []byte, port uint8, hc *HealthCheck, rate *RateLimit) *Server {

	server, exists := r.serverTable.Load(ServerNameString(name))
	if exists {
		return server
	} else {
		server := &Server{
			name:        name,
			nameString:  ServerNameString(name),
			host:        host,
			port:        port,
			status:      Offline,
			healthCheck: hc,
			rate:        rate,
		}
		r.serverTable.Store(server.nameString, server)
		return server
	}
}

func (a *FrontendApi) equal(another *FrontendApi) bool {
	if bytes.Equal(a.path, another.path) && a.pathString == another.pathString {
		return true
	} else {
		return false
	}
}

func (a *BackendApi) equal(another *BackendApi) bool {
	if bytes.Equal(a.path, another.path) && a.pathString == another.pathString {
		return true
	} else {
		return false
	}
}

func (r *Router) setStatus(status Status) {
	r.status = status
}

func (r *Router) SetOnline() {
	r.setStatus(Online)
}

func (r *Router) SetOffline() {
	r.setStatus(Offline)
}

func (r *Router) BreakDown() {
	r.setStatus(BreakDown)
}

func cmpPointerSlice(a, b []*middleware.Middleware) bool {
	var aSet, bSet mapset.Set
	aSet = mapset.NewSet()
	for _, i := range a {
		aSet.Add(i)
	}
	bSet = mapset.NewSet()
	for _, j := range b {
		bSet.Add(j)
	}
	return aSet.Equal(bSet)
}

func (r *Router) equal(another *Router) bool {
	if bytes.Equal(r.name, another.name) && r.frontendApi.equal(another.frontendApi) &&
		r.backendApi.equal(another.backendApi) && r.status == another.status && r.svr.equal(another.svr) &&
		cmpPointerSlice(r.middleware, another.middleware) {
		return true
	} else {
		return false
	}
}

func (s *Service) equal(another *Service) bool {
	if bytes.Equal(s.name, another.name) && s.nameString == another.nameString && s.srv.equal(another.srv) {
		return true
	} else {
		return false
	}
}

// check status of all server under the same service, `must` means must-condition status, return the rest server-pointer
// which not confirmed to the must-condition
func (s *Service) checkServerStatus(must Status) (confirm []*Server, rest []*Server) {
	s.srv.Range(func(key ServerNameString, value *Server) {
		if value.status != must {
			rest = append(rest, value)
		} else {
			confirm = append(confirm, value)
		}
	})
	return confirm, rest
}

func (s *Server) equal(another *Server) bool {
	if bytes.Equal(s.name, another.name) && s.nameString == another.nameString && s.status == another.status &&
		bytes.Equal(s.host, another.host) && s.port == another.port {
		return true
	} else {
		return false
	}
}
