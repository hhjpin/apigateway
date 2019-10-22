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
package routing

import (
	"bytes"
	"container/ring"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/golang/time/rate"
	"strconv"
)

const (
	Offline Status = iota
	Online
	BreakDown
)

var (
	VariableIdentifier = []byte(":")
	AnyMatchIdentifier = []byte("*")
	UriSlash           = []byte("/")
)

type Status uint8
type FrontendApiString string
type BackendApiString string
type EndpointNameString string
type ServiceNameString string
type RouterNameString string

// Routing table struct, should not be copied at any time. Using function `Init()` to create it
type Table struct {
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

	cli *clientv3.Client
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
	id         string
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

func (s Status) String() string {
	return strconv.FormatInt(int64(s), 10)
}

func (r *Table) addBackendEndpoint(ep *Endpoint) (ok bool, err error) {

	_, exists := r.endpointTable.Load(ep.nameString)
	if exists {
		logger.Info("ep already exists")
		return false, errors.New(122)
	}
	r.endpointTable.Store(ep.nameString, ep)
	return true, nil
}

func (r *Table) endpointExists(ep *Endpoint) bool {
	_, exists := r.endpointTable.Load(ep.nameString)
	return exists
}

// check input endpoint exist in the endpoint-table or not, return the rest not-existed endpoint slice
func (r *Table) endpointSliceExists(ep []*Endpoint) (rest []*Endpoint) {
	if len(ep) > 0 {
		for _, s := range ep {
			if !r.endpointExists(s) {
				rest = append(rest, s)
			}
		}
	}
	return rest
}

func (r *Table) GetRouterByName(name []byte) (*Router, error) {
	router, exists := r.routerTable.Load(RouterNameString(name))
	if !exists {
		logger.Warningf("can not find router by name: %s", RouterNameString(name))
		return nil, errors.New(131)
	}
	return router, nil
}

func (r *Table) GetServiceByName(name []byte) (*Service, error) {
	svr, exists := r.serviceTable.Load(ServiceNameString(name))
	if !exists {
		logger.Warningf("can not find service by name: %s", ServiceNameString(name))
		return nil, errors.New(137)
	}
	return svr, nil
}

func (r *Table) GetEndpointByName(name []byte) (*Endpoint, error) {
	ep, exists := r.endpointTable.Load(EndpointNameString(name))
	if !exists {
		logger.Warningf("can not find endpoint by name: %s", EndpointNameString(name))
		return nil, errors.New(139)
	}
	return ep, nil
}

func (r *Table) GetEndpointById(id string) (*Endpoint, bool) {
	var ep *Endpoint
	r.endpointTable.Range(func(key EndpointNameString, value *Endpoint) bool {
		if value.id == id {
			ep = value
			return true
		}
		return false
	})
	if ep == nil {
		return nil, false
	}
	return ep, true
}

func (r *Table) RemoveRouter(router *Router) (ok bool, err error) {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		logger.Warning("router not exists")
		return false, errors.New(125)
	}

	if router.status == Online {
		// router is online, router can not be unregistered
		return false, errors.New(132)
	}

	r.table.Delete(router.frontendApi.pathString)

	return true, nil
}

func (r *Table) SetRouterOnline(router *Router) (ok bool, err error) {

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		logger.Warning("router not exists")
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
						logger.Error("error raised when add endpoint to endpoint-table")
						return false, errors.New(134)
					}
				}
			}
			if _, err := utils.PutKV(r.cli, router.key(constant.StatusKeyString), Online.String()); err != nil {
				logger.Exception(err)
				return false, err
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
		return true, nil
	}
}

func (r *Table) SetRouterStatus(router *Router, status Status) (ok bool, err error) {
	if status == Online {
		return r.SetRouterOnline(router)
	}

	_, exists := r.table.Load(router.frontendApi.pathString)
	if !exists {
		logger.Warning("router not exists")
		return false, errors.New(125)
	}

	_, exists = r.onlineTable.Load(router.frontendApi)
	if !exists {
		// not exist in online table, do nothing
	} else {
		r.onlineTable.Delete(router.frontendApi)
	}
	if _, err := utils.PutKV(r.cli, router.key(constant.StatusKeyString), status.String()); err != nil {
		logger.Exception(err)
		return false, err
	}
	router.setStatus(status)
	return true, nil
}

func (r *Table) SetEndpointOnline(ep *Endpoint) error {
	_, exists := r.endpointTable.Load(ep.nameString)
	if !exists {
		logger.Warning("endpoint not exists")
		return errors.New(139)
	}
	resp, err := utils.GetKV(r.cli, ep.key(constant.FailedTimesKeyString))
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if resp.Count == 0 {
		//
	} else {
		if _, err := utils.PutKV(r.cli, ep.key(constant.FailedTimesKeyString), "0"); err != nil {
			logger.Exception(err)
		}
	}
	if _, err := utils.PutKV(r.cli, ep.key(constant.StatusKeyString), Online.String()); err != nil {
		logger.Exception(err)
		return err
	}
	ep.setStatus(Online)
	return nil
}

func (r *Table) SetEndpointStatus(ep *Endpoint, status Status) error {
	_, exists := r.endpointTable.Load(ep.nameString)
	if !exists {
		logger.Warning("endpoint not exists")
		return errors.New(139)
	}
	switch status {
	case Online:
		return r.SetEndpointOnline(ep)
	case BreakDown:
		var failedTimes int64
		resp, err := utils.GetKV(r.cli, ep.key(constant.FailedTimesKeyString))
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		if resp.Count == 0 {
			logger.Debugf("no failed times key")
			if _, err := utils.PutKV(r.cli, ep.key(constant.FailedTimesKeyString), "1"); err != nil {
				logger.Exception(err)
			}
		} else {
			failedTimes, err = strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
			if err != nil {
				logger.Exception(err)
				failedTimes = 1
			}
			if _, err := utils.PutKV(r.cli, ep.key(constant.FailedTimesKeyString), strconv.FormatInt(failedTimes+1, 10)); err != nil {
				logger.Exception(err)
			}
		}
		logger.Debugf("健康检查重试 failedTimes: %d, maxRetryTimes: %d", int(failedTimes), int(ep.healthCheck.retryTime))
		if int(failedTimes) >= int(ep.healthCheck.retryTime) {
			// exceed max retry times, tag this ep to offline
			if _, err := utils.PutKV(r.cli, ep.key(constant.StatusKeyString), Offline.String()); err != nil {
				logger.Exception(err)
				return err
			}
			ep.setStatus(Offline)
		} else {
			if _, err := utils.PutKV(r.cli, ep.key(constant.StatusKeyString), BreakDown.String()); err != nil {
				logger.Exception(err)
				return err
			}
			ep.setStatus(BreakDown)
		}
		return nil
	case Offline:
		if _, err := utils.PutKV(r.cli, ep.key(constant.StatusKeyString), status.String()); err != nil {
			logger.Exception(err)
			return err
		}
		ep.setStatus(status)
		return nil
	default:
		logger.Warningf("unrecognized status: %s", status.String())
		return nil
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

func (r *Router) equal(another *Router) bool {
	if bytes.Equal(r.name, another.name) && r.frontendApi.equal(another.frontendApi) &&
		r.backendApi.equal(another.backendApi) && r.status == another.status && r.service.equal(another.service) &&
		utils.CmpPointerSlice(r.middleware, another.middleware) {
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

func (r *Router) key(attr ...string) string {
	if len(attr) > 0 {
		return constant.RouterDefinition + fmt.Sprintf(constant.RouterPrefixString, string(r.name)) + attr[0]
	} else {
		return constant.RouterDefinition + fmt.Sprintf(constant.RouterPrefixString, string(r.name))
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
	s.ep.Range(func(key EndpointNameString, value *Endpoint) bool {
		if value.status != must {
			rest = append(rest, value)
		} else {
			confirm = append(confirm, value)
		}
		return false
	})
	return confirm, rest
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

func (ep *Endpoint) key(attr ...string) string {
	if len(attr) > 0 {
		return constant.NodeDefinition + fmt.Sprintf(constant.NodePrefixString, ep.id) + attr[0]
	} else {
		return constant.NodeDefinition + fmt.Sprintf(constant.NodePrefixString, ep.id)
	}
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

func (r *Table) Select(input []byte, method []byte) (TargetServer, error) {
	var replacedBackendUri []byte
	var matchRouter *Router

	input = []byte(string(method) + "@" + string(input))
	inputByteSlice := bytes.Split(input, UriSlash)
	r.onlineTable.Range(func(key *FrontendApi, value *Router) bool {
		matched, replaced := match(inputByteSlice, key.pattern, value.backendApi.pattern)
		if matched && value.status == Online {
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
		next := matchRouter.service.onlineEp.Value
		matchRouter.service.onlineEp = matchRouter.service.onlineEp.Move(1)
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

func match(input, pattern, backend [][]byte) (bool, []byte) {
	inputLen := len(input)
	patternLen := len(pattern)
	if inputLen >= patternLen {
		var target [][]byte
		var hasAny bool
		var replaced = make(map[string][]byte)
		for i := 0; i < patternLen; i++ {
			if bytes.HasPrefix(pattern[i], AnyMatchIdentifier) {
				replaced[string(pattern[i])] = bytes.Join(input[i:], UriSlash)
				hasAny = true
				break
			} else if bytes.HasPrefix(pattern[i], VariableIdentifier) {
				replaced[string(pattern[i])] = input[i]
			} else if !bytes.Equal(pattern[i], input[i]) {
				return false, nil
			}
		}
		if !hasAny && inputLen != patternLen {
			return false, nil
		}
		for _, b := range backend {
			if r, ok := replaced[string(b)]; ok {
				target = append(target, r)
			} else {
				target = append(target, b)
			}
		}
		return true, bytes.Join(target, UriSlash)
	}
	return false, nil
}
