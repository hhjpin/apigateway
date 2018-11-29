/*
	Defined all component made up the routing table.
	Component:
		FrontendApi
		Router
		Service
		Endpoint
	Relation between component:
		Router:
			Front FrontendApi
			Backend FrontendApi
			Service
		Service:
			[]Endpoint
*/
package core

import (
	"bytes"
	"container/ring"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"git.henghajiang.com/backend/golang_utils/errors"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/deckarep/golang-set"
	"github.com/golang/time/rate"
	"strconv"
)

const (
	Offline Status = iota
	Online
	BreakDown
)

var (
	VariableIdentifier = []byte("$")
	UriSlash           = []byte("/")

	rtLogger = log.New()
)

type Status uint8
type FrontendApiString string
type BackendApiString string
type EndpointNameString string
type ServiceNameString string
type RouterNameString string

// Routing table struct, should not be copied at any time. Using function `Init()` to create it
type RoutingTable struct {
	Version string

	// frontend-api/router mapping table
	table ApiRouterTableMap
	// online frontend-api/router mapping table
	onlineTable OnlineApiRouterTableMap
	// service table
	serviceTable ServiceTableMap
	// endpoint table
	endpointTable EndpointTableMap
	// router table
	routerTable RouterTableMap
}

// FrontendApi-struct used to create Frontend-FrontendApi or Backend-FrontendApi. <string> type attr `pathString` is using on mapping tables
type FrontendApi struct {
	path []byte
	// string type of path variable, used for indexing mapping table
	pathString FrontendApiString
	// []byte pattern
	pattern [][]byte
}

type BackendApi struct {
	path       []byte
	pathString BackendApiString
	pattern    [][]byte
}

// Endpoint-struct defined a backend-endpoint
type Endpoint struct {
	name       []byte
	nameString EndpointNameString

	host   []byte
	port   int
	status Status // 0 -> offline, 1 -> online, 2 -> breakdown

	healthCheck *HealthCheck
	rate        *rate.Limiter
}

// Service-struct defined a backend-service
type Service struct {
	name       []byte
	nameString ServiceNameString
	ep         *EndpointTableMap
	onlineEp   *ring.Ring

	// if request method not in the accept http method slice, return HTTP 405
	// if AcceptHttpMethod slice is empty, allow all http verb.
	acceptHttpMethod [][]byte
}

type Router struct {
	name   []byte
	status Status // 0 -> offline, 1 -> online, 2 -> breakdown

	frontendApi *FrontendApi
	backendApi  *BackendApi
	service     *Service

	middleware []*middleware.Middleware
}

type TargetServer struct {
	host []byte
	uri  []byte
	svr  []byte
}

func (r *RoutingTable) addBackendService(service *Service) (ok bool, err error) {

	_, exists := r.serviceTable.Load(service.nameString)
	if exists {
		rtLogger.Info("service already exists")
		return false, errors.New(120)
	}
	r.serviceTable.Store(service.nameString, service)
	return true, nil
}

func (r *RoutingTable) removeBackendService(service *Service) (ok bool, err error) {

	_, exists := r.serviceTable.Load(service.nameString)
	if !exists {
		rtLogger.Info("service not exist")
		return false, errors.New(121)
	}
	r.serviceTable.Delete(service.nameString)
	return true, nil
}

func (r *RoutingTable) addBackendEndpoint(ep *Endpoint) (ok bool, err error) {

	_, exists := r.endpointTable.Load(ep.nameString)
	if exists {
		rtLogger.Info("ep already exists")
		return false, errors.New(122)
	}
	r.endpointTable.Store(ep.nameString, ep)
	return true, nil
}

func (r *RoutingTable) removeBackendEndpoint(ep *Endpoint) (ok bool, err error) {

	_, exists := r.endpointTable.Load(ep.nameString)
	if !exists {
		rtLogger.Info("ep not exist")
		return false, errors.New(123)
	}
	r.endpointTable.Delete(ep.nameString)
	return true, nil
}

func (r *RoutingTable) endpointExists(ep *Endpoint) bool {
	_, exists := r.endpointTable.Load(ep.nameString)
	return exists
}

// check input endpoint exist in the endpoint-table or not, return the rest not-existed endpoint slice
func (r *RoutingTable) endpointSliceExists(ep []*Endpoint) (rest []*Endpoint) {
	if len(ep) > 0 {
		for _, s := range ep {
			if !r.endpointExists(s) {
				rest = append(rest, s)
			}
		}
	}
	return rest
}

func (r *RoutingTable) CreateBackendApi(path []byte) *BackendApi {
	path = prefixFilter(path)
	return &BackendApi{
		path:       path,
		pathString: BackendApiString(path),
		pattern:    bytes.Split(path, UriSlash),
	}
}

// create frontend api obj, return pointer of api obj. if already exists, return that one
func (r *RoutingTable) CreateFrontendApi(path []byte) (api *FrontendApi, ok bool) {
	path = prefixFilter(path)
	if router, exists := r.table.Load(FrontendApiString(path)); exists {
		ok = false
		api = router.frontendApi
	} else {
		ok = true
		api = &FrontendApi{
			path:       path,
			pathString: FrontendApiString(path),
			pattern:    bytes.Split(path, UriSlash),
		}
	}
	return api, ok
}

func (r *RoutingTable) RemoveFrontendApi(path []byte) (ok bool, err error) {
	if router, exists := r.table.Load(FrontendApiString(path)); exists {
		if router.status == Online {
			// router is online, api can not be deleted
			return false, errors.New(124)
		} else {
			// remove frontend api from router obj.
			router.frontendApi = nil
			return true, nil
		}
	} else {
		return false, errors.New(125)
	}
}

func (r *RoutingTable) CreateRouter(name []byte, fApi *FrontendApi, bApi *BackendApi, svr *Service, mw ...*middleware.Middleware) error {
	if fApi.pathString == "" || bApi.pathString == "" {
		rtLogger.Error("api obj not completed")
		return errors.New(126)
	}
	if svr.nameString == "" {
		rtLogger.Error("service obj not completed")
		return errors.New(127)
	}

	_, exists := r.table.Load(fApi.pathString)

	if exists {
		return errors.New(128)
	} else {
		router := &Router{
			name:        name,
			status:      Offline,
			frontendApi: fApi,
			backendApi:  bApi,
			service:     svr,
			middleware:  mw,
		}

		onlineEndpoint, _ := svr.checkEndpointStatus(Online)
		if len(onlineEndpoint) > 0 {
			rest := r.endpointSliceExists(onlineEndpoint)
			if len(rest) > 0 {
				for _, i := range rest {
					_, e := r.addBackendEndpoint(i)
					if e != nil {
						rtLogger.Error("error raised when add endpoint to endpoint-table")
						return errors.New(129)
					}
				}
			}
		} else {
			return errors.New(130)
		}
		r.addBackendService(svr)
		r.table.Store(fApi.pathString, router)
		r.routerTable.Store(RouterNameString(router.name), router)
		return nil
	}
}

func (r *RoutingTable) GetRouterByName(name []byte) (*Router, error) {
	router, exists := r.routerTable.Load(RouterNameString(name))
	if !exists {
		rtLogger.Warning("can not find router by name")
		return nil, errors.New(131)
	}
	return router, nil
}

func (r *RoutingTable) RemoveRouter(router *Router) (ok bool, err error) {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		rtLogger.Warning("router not exists")
		return false, errors.New(125)
	}

	if router.status == Online {
		// router is online, router can not be unregistered
		return false, errors.New(132)
	}

	r.table.Delete(router.frontendApi.pathString)

	return true, nil
}

func (r *RoutingTable) SetRouterOnline(router *Router) (ok bool, err error) {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		rtLogger.Warning("router not exists")
		return false, errors.New(125)
	}

	onlineRouter, exists := r.onlineTable.Load(router.frontendApi)
	if !exists {
		// not exist in online table
		// check backend endpoint status first
		onlineEndpoint, _ := router.service.checkEndpointStatus(Online)
		if len(onlineEndpoint) == 0 {
			// all endpoint are not online, this router can not be set to online
			return false, errors.New(133)
		} else {
			rest := r.endpointSliceExists(onlineEndpoint)
			if len(rest) > 0 {
				// some member in <slice> onlineEndpoint is not exist in r.endpointTable
				// this will happen at data-error
				for _, i := range rest {
					_, e := r.addBackendEndpoint(i)
					if e != nil {
						rtLogger.Error("error raised when add endpoint to endpoint-table")
						return false, errors.New(134)
					}
				}
			}
			router.setStatus(Online)
			r.onlineTable.Store(router.frontendApi, router)
			return true, nil
		}
	} else {
		// exist in online table
		// check router == onlineRouter
		if router != onlineRouter {
			// data mapping error
			return false, errors.New(135)
		}
		return false, errors.New(136)
	}
}

func (r *RoutingTable) SetRouterStatus(router *Router, status Status) (ok bool, err error) {
	if status == Online {
		return r.SetRouterOnline(router)
	}

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		rtLogger.Warning("router not exists")
		return false, errors.New(125)
	}

	_, exists = r.onlineTable.Load(router.frontendApi)
	if !exists {
		// not exist in online table, do nothing
	} else {
		r.onlineTable.Delete(router.frontendApi)
	}
	router.setStatus(status)
	return true, nil
}

func (r *RoutingTable) CreateService(name []byte, acceptHttpMethod [][]byte) *Service {

	svr, exists := r.serviceTable.Load(ServiceNameString(name))
	if exists {
		return svr
	} else {
		service := &Service{
			name:             name,
			nameString:       ServiceNameString(name),
			ep:               &EndpointTableMap{},
			acceptHttpMethod: acceptHttpMethod,
		}
		r.serviceTable.Store(service.nameString, service)
		return service
	}
}

func (r *RoutingTable) GetServiceByName(name []byte) (*Service, error) {

	service, exists := r.serviceTable.Load(ServiceNameString(name))
	if !exists {
		rtLogger.Warning("can not find service by name")
		return nil, errors.New(137)
	}
	return service, nil
}

func (r *RoutingTable) RemoveService(svr *Service) error {
	var err error

	_, exists := r.serviceTable.Load(svr.nameString)
	if !exists {
		rtLogger.Warning("can not find service by name")
		return errors.New(137)
	}

	r.table.Lock()
	r.table.unsafeRange(func(key FrontendApiString, value *Router) {
		if value.service == svr {
			if value.status == Online {
				err = errors.New(138)
				return
			}
		}
	})

	r.table.unsafeRange(func(key FrontendApiString, value *Router) {
		if value.service == svr {
			value.service = nil
		}
	})
	r.table.Unlock()

	r.serviceTable.Delete(svr.nameString)
	return err
}

func (r *RoutingTable) CreateEndpoint(name, host []byte, port int, hc *HealthCheck, rate *rate.Limiter) *Endpoint {

	endpoint, exists := r.endpointTable.Load(EndpointNameString(name))
	if exists {
		return endpoint
	} else {
		endpoint := &Endpoint{
			name:        name,
			nameString:  EndpointNameString(name),
			host:        host,
			port:        port,
			status:      Offline,
			healthCheck: hc,
			rate:        rate,
		}
		r.endpointTable.Store(endpoint.nameString, endpoint)
		return endpoint
	}
}

func (r *RoutingTable) GetEndpointByName(name EndpointNameString) (*Endpoint, error) {

	endpoint, exists := r.endpointTable.Load(name)
	if exists {
		return endpoint, nil
	} else {
		rtLogger.Warning("can not find endpoint by name")
		return nil, errors.New(139)
	}
}

func (r *RoutingTable) SetEndpointOnline(ep *Endpoint) error {
	_, exists := r.endpointTable.Load(ep.nameString)
	if !exists {
		rtLogger.Warning("endpoint not exists")
		return errors.New(139)
	}
	ep.setStatus(Online)
	return nil
}

func (r *RoutingTable) RemoveEndpoint(svr *Endpoint) error {

	_, exists := r.endpointTable.Load(svr.nameString)
	if !exists {
		rtLogger.Warning("can not find endpoint by name")
		return errors.New(139)
	}

	r.serviceTable.Lock()
	r.serviceTable.unsafeRange(func(key ServiceNameString, value *Service) {
		value.ep.Lock()
		value.ep.unsafeRange(func(k EndpointNameString, v *Endpoint) {
			if v == svr {
				value.ep.Delete(svr.nameString)
			}
		})
		value.ep.Unlock()
	})
	r.serviceTable.Unlock()

	r.endpointTable.Delete(svr.nameString)
	return nil
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
		r.backendApi.equal(another.backendApi) && r.status == another.status && r.service.equal(another.service) &&
		cmpPointerSlice(r.middleware, another.middleware) {
		return true
	} else {
		return false
	}
}

func (r *Router) CheckStatus(must Status) bool {
	confirm, _ := r.service.checkEndpointStatus(must)
	if len(confirm) == 0 {
		// not online
		return false
	} else {
		return true
	}
}

func (s *Service) equal(another *Service) bool {
	if bytes.Equal(s.name, another.name) && s.nameString == another.nameString && s.ep.equal(another.ep) {
		return true
	} else {
		return false
	}
}

// check status of all endpoint under the same service, `must` means must-condition status, return the rest endpoint
// which not confirmed to the must-condition
func (s *Service) checkEndpointStatus(must Status) (confirm []*Endpoint, rest []*Endpoint) {
	s.ep.Range(func(key EndpointNameString, value *Endpoint) {
		if value.status != must {
			rest = append(rest, value)
		} else {
			confirm = append(confirm, value)
		}
	})
	return confirm, rest
}

func (s *Service) AddEndpoint(ep *Endpoint) (bool, error) {
	another, exists := s.ep.Load(ep.nameString)
	if exists {
		if ep.equal(another) {
			return true, nil
		} else {
			return false, errors.New(122)
		}
	} else {
		if ep.status == Online {
			newNode := ring.New(1)
			newNode.Value = ep.nameString
			n := s.onlineEp.Link(newNode)
			newNode.Link(n)
		}
		s.ep.Store(ep.nameString, ep)

		return true, nil
	}
}

func (s *Service) ResetOnlineEndpointRing(online []*Endpoint) error {
	if len(online) == 0 {
		return errors.New(144)
	}
	for idx, ep := range online {
		newNode := ring.New(1)
		newNode.Value = ep.nameString
		if idx == 0 {
			s.onlineEp = newNode
		} else {
			r := s.onlineEp.Link(newNode)
			newNode.Link(r)
		}
	}
	return nil
}

func (ep *Endpoint) equal(another *Endpoint) bool {
	if bytes.Equal(ep.name, another.name) && ep.nameString == another.nameString && ep.status == another.status &&
		bytes.Equal(ep.host, another.host) && ep.port == another.port {
		return true
	} else {
		return false
	}
}

func (ep *Endpoint) setStatus(status Status) {
	ep.status = status
}

func (r *RoutingTable) Select(input []byte) (TargetServer, error) {
	var replacedBackendUri []byte
	var matchRouter *Router

	inputByteSlice := bytes.Split(input, UriSlash)
	r.onlineTable.Range(func(key *FrontendApi, value *Router) bool {
		matched, replaced := match(inputByteSlice, key.pattern, value.backendApi.pattern)
		if matched {
			matchRouter = value
			replacedBackendUri = replaced
			return true
		}
		return false
	})

	if matchRouter == nil {
		return TargetServer{}, errors.New(142)
	}
	if matchRouter.status != Online {
		return TargetServer{}, errors.New(143)
	}

	ringLength := matchRouter.service.onlineEp.Len()
	counts := 0
	for {
		next := matchRouter.service.onlineEp.Next().Value
		_, ok := next.(EndpointNameString)
		if !ok {
			return TargetServer{}, errors.New(140)
		}
		ep, exists := matchRouter.service.ep.Load(next.(EndpointNameString))
		if !exists {
			return TargetServer{}, errors.New(123)
		}
		if ep.status == Online {
			return TargetServer{
				host: bytes.Join([][]byte{ep.host, []byte(strconv.FormatInt(int64(ep.port), 10))}, []byte(":")),
				uri:  replacedBackendUri,
				svr:  matchRouter.service.name,
			}, nil
		}
		ringLength = matchRouter.service.onlineEp.Len()
		counts++
		if counts > ringLength {
			break
		}
	}
	return TargetServer{}, errors.New(141)
}

func prefixFilter(input []byte) []byte {
	ret := bytes.TrimPrefix(input, UriSlash)
	if bytes.HasPrefix(ret, UriSlash) {
		ret = prefixFilter(ret)
	}
	ret = bytes.TrimSuffix(ret, UriSlash)
	if bytes.HasSuffix(ret, UriSlash) {
		ret = prefixFilter(ret)
	}
	return ret
}

func match(input, pattern [][]byte, backend [][]byte) (bool, []byte) {
	inputLen := len(input)
	patternLen := len(pattern)
	if inputLen == patternLen {
		var tmp [][]byte
		replaced := make(map[string][]byte)
		for i := 0; i < patternLen; i++ {
			if bytes.HasPrefix(pattern[i], VariableIdentifier) {
				replaced[string(pattern[i])] = input[i]
			} else {
				if !bytes.Equal(pattern[i], input[i]) {
					return false, nil
				}
			}
		}
		for _, b := range backend {
			if r, ok := replaced[string(b)]; ok {
				tmp = append(tmp, r)
			} else {
				tmp = append(tmp, b)
			}
		}
		return true, bytes.Join(tmp, SlashBytes)
	} else {
		return false, nil
	}
}
