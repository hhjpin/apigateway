package routing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"github.com/coreos/etcd/clientv3"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/logger"
	"os"
	"strconv"
)

func InitRoutingTable(cli *clientv3.Client) *Table {
	var rt Table
	var epSlice []*Endpoint

	rt.cli = cli
	rt.Version = "1.0.0"
	rt.events = NewEvents()
	ol := NewOnlineRouteTableMap()
	svrMap, epMap, err := initServiceNode(cli)
	if err != nil {
		logger.Error(err)
		os.Exit(-1)
	}
	rt.serviceTable = *svrMap
	rt.endpointTable = *epMap
	rt.onlineTable = *ol
	routerTable, table, err := initRouter(cli, &rt.serviceTable)
	if err != nil {
		logger.Error(err)
	}
	rt.table = *table
	rt.routerTable = *routerTable

	rt.endpointTable.Range(func(key EndpointNameString, value *Endpoint) bool {
		if value.healthCheck.path != nil {
			epSlice = append(epSlice, value)
		}
		return false
	})
	if len(epSlice) > 0 {
		for _, ep := range epSlice {
			if check, err := ep.healthCheck.Check(ep.host, ep.port); check {
				if err = rt.SetEndpointOnline(ep); err != nil {
					ep.setStatus(Online)
				}
			} else {
				if err = rt.SetEndpointStatus(ep, BreakDown); err != nil {
					ep.setStatus(BreakDown)
				}
			}
		}
	}
	rt.routerTable.Range(func(key RouterNameString, value *Router) {
		var ok bool
		if value.CheckStatus(Online) {
			ok, _ = rt.SetRouterStatus(value, Online)
		} else {
			ok, _ = rt.SetRouterStatus(value, Offline)
		}
		if ok {
			confirm, _ := value.service.checkEndpointStatus(Online)
			if err := value.service.ResetOnlineEndpointRing(confirm); err != nil {
				logger.Error(err.(errors.Error).String(), " ", key, " ", value.frontendApi.pathString)
			}
		}
	})
	return &rt
}

func initServiceNode(cli *clientv3.Client) (*ServiceTableMap, *EndpointTableMap, error) {
	svrMap := NewServiceTableMap()
	epMap := NewEndpointTableMap()

	resp, err := utils.GetPrefixKV(cli, constant.ServiceDefinition, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return svrMap, epMap, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, constant.ServiceDefinitionBytes)
		tmp := bytes.Split(key, constant.SlashBytes)
		if len(tmp) != 2 {
			logger.Infof("invalid service definition: %s", key)
			continue
		} else {
			sName := bytes.TrimPrefix(tmp[0], constant.ServicePrefixBytes)
			if s, ok := svrMap.Load(ServiceNameString(sName)); ok {
				// service exists
				if bytes.Equal(tmp[1], constant.NameKeyBytes) {
					s.name = kv.Value
					s.nameString = ServiceNameString(kv.Value)
				} else if bytes.Equal(tmp[1], constant.NodeKeyBytes) {
					// check node
					var nodeSlice []string

					err = json.Unmarshal(kv.Value, &nodeSlice)
					if err != nil {
						logger.Error(err)
						return nil, nil, err
					}
					for _, n := range nodeSlice {
						ep, err := initEndpointNode(cli, n)
						if err != nil {
							logger.Error(err.Error())
							continue
						}
						if s.ep != nil {
							if _, ok := s.ep.Load(ep.nameString); !ok {
								s.ep.Store(ep.nameString, ep)
							}
						} else {
							tmp := NewEndpointTableMap()
							tmp.Store(ep.nameString, ep)
							s.ep = tmp
						}
					}
					s.ep.Range(func(key EndpointNameString, value *Endpoint) bool {
						epMap.Store(key, value)
						return false
					})
				} else {
					logger.Warnf("unrecognized node attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
				}
			} else {
				if bytes.Equal(tmp[1], constant.NameKeyBytes) {
					if bytes.Equal(kv.Value, sName) {
						s = &Service{
							name:       sName,
							nameString: ServiceNameString(sName),
							ep:         nil,
							onlineEp:   nil,
						}
						svrMap.Store(ServiceNameString(sName), s)
					} else {
						logger.Infof("invalid service name, key: %s", string(kv.Key))
					}
				} else if bytes.Equal(tmp[1], constant.NodeKeyBytes) {
					var nodeSlice []string
					epSlice := NewEndpointTableMap()

					s = &Service{
						name:       sName,
						nameString: ServiceNameString(sName),
						ep:         nil,
						onlineEp:   nil,
					}
					err = json.Unmarshal(kv.Value, &nodeSlice)
					if err != nil {
						logger.Error(err)
						return nil, nil, err
					}
					for _, n := range nodeSlice {
						ep, err := initEndpointNode(cli, n)
						if err != nil {
							logger.Error(err.Error())
							continue
						}
						epSlice.Store(ep.nameString, ep)
					}
					s.ep = epSlice
					svrMap.Store(s.nameString, s)
					s.ep.Range(func(key EndpointNameString, value *Endpoint) bool {
						epMap.Store(key, value)
						return false
					})
				} else {
					logger.Warnf("unrecognized node attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
				}
			}
		}
	}
	return svrMap, epMap, nil
}

func initEndpointNode(cli *clientv3.Client, nodeID string) (*Endpoint, error) {
	var ep Endpoint

	resp, err := utils.GetPrefixKV(cli, constant.NodePrefixDefinition+nodeID, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(constant.NodePrefixDefinition+nodeID+constant.Slash))
		if bytes.Equal(key, constant.IdKeyBytes) {
			ep.id = string(kv.Value)
		} else if bytes.Equal(key, constant.NameKeyBytes) {
			ep.name = kv.Value
			ep.nameString = EndpointNameString(kv.Value)
		} else if bytes.Equal(key, constant.HostKeyBytes) {
			ep.host = kv.Value
		} else if bytes.Equal(key, constant.PortKeyBytes) {
			tmpInt, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			ep.port = int(tmpInt)
		} else if bytes.Equal(key, constant.StatusKeyBytes) {
			tmpInt, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			switch Status(uint8(tmpInt)) {
			case Offline:
				ep.status = Offline
			case Online:
				ep.status = Online
			case BreakDown:
				ep.status = BreakDown
			default:
				return nil, errors.New(150)
			}
		} else if bytes.Equal(key, []byte(constant.FailedTimesKeyString)) {
			// do nothing
		} else if bytes.Equal(key, constant.HealthCheckKeyBytes) {
			// get health check info
			respA, err := utils.GetPrefixKV(cli, constant.HealthCheckPrefixDefinition+string(kv.Value), clientv3.WithPrefix())
			if err != nil {
				logger.Error(err)
				return nil, err
			}

			var hc HealthCheck
			hc.path = nil
			for _, kvA := range respA.Kvs {
				keyA := bytes.TrimPrefix(kvA.Key, []byte(constant.HealthCheckPrefixDefinition+string(kv.Value)+constant.Slash))
				if bytes.Equal(keyA, constant.IdKeyBytes) {
					hc.id = string(kvA.Value)
				} else if bytes.Equal(keyA, constant.PathKeyBytes) {
					hc.path = kvA.Value
				} else if bytes.Equal(keyA, constant.TimeoutKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Error(err)
						return nil, err
					}
					hc.timeout = uint8(tmpInt)
				} else if bytes.Equal(keyA, constant.IntervalKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Error(err)
						return nil, err
					}
					hc.interval = uint8(tmpInt)
				} else if bytes.Equal(keyA, constant.RetryKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Error(err)
						return nil, err
					}
					if tmpInt == 0 {
						hc.retry = false
					} else {
						hc.retry = true
					}
				} else if bytes.Equal(keyA, constant.RetryTimeKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Error(err)
						return nil, err
					}
					hc.retryTime = uint8(tmpInt)
				} else {
					// unrecognized attribute
					logger.Warnf("unrecognized health check attribute, key: %s, value: %s", string(kvA.Key), string(kvA.Value))
				}
			}
			ep.healthCheck = &hc
		} else {
			logger.Warnf("unrecognized node attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
		}
	}

	if ep.nameString == "" || ep.host == nil || ep.port == 0 {
		logger.Errorf("endpoint initialized failed. uncompleted attribute assigned. %+v", ep)
		return nil, errors.New(151)
	}
	return &ep, nil
}

func initRouter(cli *clientv3.Client, svrMap *ServiceTableMap) (*RouterTableMap, *ApiRouterTableMap, error) {
	rtMap := NewRouteTableMap()
	artMap := NewApiRouterTableMap()

	resp, err := utils.GetPrefixKV(cli, constant.RouterDefinition, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, constant.RouterDefinitionBytes)
		tmpSlice := bytes.Split(key, constant.SlashBytes)
		if len(tmpSlice) != 2 {
			logger.Warnf("invalid router definition: %s", key)
			continue
		} else {
			rName := bytes.TrimPrefix(tmpSlice[0], constant.RouterPrefixBytes)
			attr := tmpSlice[1]
			r, ok := rtMap.Load(RouterNameString(rName))
			if !ok {
				r = &Router{}
				rtMap.Store(RouterNameString(rName), r)
			}
			if bytes.Equal(attr, constant.IdKeyBytes) {
				// do nothing
			} else if bytes.Equal(attr, constant.NameKeyBytes) {
				if !bytes.Equal(rName, kv.Value) {
					logger.Warnf("inconsistent router definition: %s %s", string(kv.Key), string(kv.Value))
					continue
				}
				r.name = kv.Value
			} else if bytes.Equal(attr, constant.FrontendApiKeyBytes) {
				r.frontendApi = &FrontendApi{
					path:       kv.Value,
					pathString: FrontendApiString(kv.Value),
					pattern:    bytes.Split(kv.Value, constant.SlashBytes),
				}
			} else if bytes.Equal(attr, constant.BackendApiKeyBytes) {
				r.backendApi = &BackendApi{
					path:       kv.Value,
					pathString: BackendApiString(kv.Value),
					pattern:    bytes.Split(kv.Value, constant.SlashBytes),
				}
			} else if bytes.Equal(attr, constant.ServiceKeyBytes) {
				if svr, ok := svrMap.Load(ServiceNameString(kv.Value)); ok {
					r.service = svr
				} else {
					logger.Errorf("service not exist")
					continue
				}
			} else if bytes.Equal(attr, constant.StatusKeyBytes) {
				// do nothing
			} else {
				logger.Warnf("unrecognized health check attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
			}
		}
	}

	rtMap.Range(func(key RouterNameString, value *Router) {
		if value.frontendApi == nil {
			delete(rtMap.internal, key)
			logger.Errorf("route %s: not have total attr: frontendApi", key)
			return
		}
		artMap.Store(value.frontendApi.pathString, value)
	})
	return rtMap, artMap, nil
}

func (r *Table) CreateRouter(name string, key string) error {
	resp, err := utils.GetPrefixKV(r.cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return err
	}
	router := &Router{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid router attribute key")
			return errors.NewFormat(200, "invalid router attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.IdKeyString:
		case constant.FrontendApiKeyString:
			router.frontendApi = &FrontendApi{
				path:       kv.Value,
				pathString: FrontendApiString(kv.Value),
				pattern:    bytes.Split(kv.Value, constant.SlashBytes),
			}
		case constant.BackendApiKeyString:
			router.backendApi = &BackendApi{
				path:       kv.Value,
				pathString: BackendApiString(kv.Value),
				pattern:    bytes.Split(kv.Value, constant.SlashBytes),
			}
		case constant.NameKeyString:
			router.name = kv.Value
		case constant.ServiceKeyString:
			if svr, err := r.GetServiceByName(kv.Value); err != nil {
				// no service
				router.service = &Service{
					name:             kv.Value,
					nameString:       ServiceNameString(kv.Value),
					ep:               NewEndpointTableMap(),
					onlineEp:         nil,
					acceptHttpMethod: nil,
				}
			} else {
				router.service = svr
			}
		case constant.StatusKeyString:
		default:
			logger.Errorf("unsupported router attribute: %s", keyStr)
			return errors.NewFormat(200, fmt.Sprintf("unsupported router attribute: %s", keyStr))
		}
	}

	if router.frontendApi == nil {
		logger.Warnf("router %s frontendApi is not ready", name)
		return errors.New(126)
	}

	r.table.Store(router.frontendApi.pathString, router)
	r.routerTable.Store(RouterNameString(router.name), router)
	confirm, _ := router.service.checkEndpointStatus(Online)
	if len(confirm) > 0 {
		if err := router.service.ResetOnlineEndpointRing(confirm); err != nil {
			logger.Error(err)
			return err
		}
		if _, err := r.SetRouterOnline(router); err != nil {
			logger.Error(err)
			return err
		}
	}
	return nil
}

//not use in routerTable
func (r *Table) RefreshRouterByName(name string, key string) error {
	router, err := r.GetRouterByName([]byte(name))
	if err != nil {
		// can not find router in routing table, try to generate a new one
		return r.CreateRouter(name, key)
	}
	return r.RefreshRouter(router, key)
}

func (r *Table) RefreshRouter(router *Router, key string) error {
	resp, err := utils.GetPrefixKV(r.cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return err
	}
	if router.CheckStatus(Online) {
		// if router is online now, it will not be refreshed
		// TODO: trigger alarm

		//return errors.New(132)
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid router attribute key")
			return errors.NewFormat(200, "invalid router attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.IdKeyString:
		case constant.FrontendApiKeyString:
			if !bytes.Equal(router.frontendApi.path, kv.Value) {
				tmp := router.frontendApi.pathString
				router.frontendApi.path = kv.Value
				router.frontendApi.pathString = FrontendApiString(kv.Value)
				router.frontendApi.pattern = bytes.Split(kv.Value, constant.SlashBytes)
				r.table.Store(FrontendApiString(kv.Value), router)
				r.table.Delete(tmp)
			}
		case constant.BackendApiKeyString:
			router.backendApi.path = kv.Value
			router.backendApi.pathString = BackendApiString(kv.Value)
			router.backendApi.pattern = bytes.Split(kv.Value, constant.SlashBytes)
		case constant.NameKeyString:
			router.name = kv.Value
		case constant.ServiceKeyString:
			if svr, err := r.GetServiceByName(kv.Value); err != nil {
				// no service
				router.service = &Service{
					name:             kv.Value,
					nameString:       ServiceNameString(kv.Value),
					ep:               NewEndpointTableMap(),
					onlineEp:         nil,
					acceptHttpMethod: nil,
				}
			} else {
				router.service = svr
			}
		case constant.StatusKeyString:
		default:
			logger.Errorf("unsupported router attribute: %s", keyStr)
			return errors.NewFormat(200, fmt.Sprintf("unsupported router attribute: %s", keyStr))
		}
	}
	confirm, _ := router.service.checkEndpointStatus(Online)
	if len(confirm) > 0 {
		if err := router.service.ResetOnlineEndpointRing(confirm); err != nil {
			logger.Error(err)
			return err
		}
		if _, err := r.SetRouterOnline(router); err != nil {
			logger.Error(err)
			return err
		}
	}
	return nil
}

func (r *Table) DeleteRouter(name string) error {
	router, err := r.GetRouterByName([]byte(name))
	if err != nil {
		// router already deleted
		return nil
	}
	r.table.Delete(router.frontendApi.pathString)
	r.routerTable.Delete(RouterNameString(router.name))
	r.onlineTable.Delete(router.frontendApi)
	logger.Debugf("route delete successful: %s", name)
	return nil
}

func (r *Table) CreateService(name string, key string) error {
	resp, err := utils.GetPrefixKV(r.cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return err
	}
	svr := &Service{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid service attribute key")
			return errors.NewFormat(200, "invalid service attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.NodeKeyString:
			var nodeSlice []string
			if err := json.Unmarshal(kv.Value, &nodeSlice); err != nil {
				logger.Error(err)
				return err
			}
			svr.ep = NewEndpointTableMap()
			for _, node := range nodeSlice {
				if ep, err := r.GetEndpointByName([]byte(node)); err != nil {
					// endpoint not exists in memory, maybe process it later
					logger.Debugf("endpoint [%s] not exists in memory, maybe process it later", node)
				} else {
					svr.ep.Store(ep.nameString, ep)
				}
			}
			confirm, _ := svr.checkEndpointStatus(Online)
			if len(confirm) > 0 {
				if err := svr.ResetOnlineEndpointRing(confirm); err != nil {
					logger.Error(err)
				}
			}
		case constant.NameKeyString:
			svr.name = kv.Value
			svr.nameString = ServiceNameString(kv.Value)
		default:
			logger.Errorf("unsupported service attribute: %s", keyStr)
			return errors.NewFormat(200, fmt.Sprintf("unsupported service attribute: %s", keyStr))
		}
	}
	r.serviceTable.Store(svr.nameString, svr)
	r.routerTable.Range(func(key RouterNameString, value *Router) {
		if value.service.nameString == svr.nameString {
			// service connected
			logger.Debugf("current router service: %+v", value.service)
			value.service = svr
			if status := value.CheckStatus(Online); status {
				value.setStatus(Online)
				if _, ok := r.onlineTable.Load(value.frontendApi); ok {
					// router has no available service but exists in online api table
					logger.Warnf("router has no available service but exists in online api table")
				} else {
					r.onlineTable.Store(value.frontendApi, value)
				}
			}
		}
	})
	return nil
}

//not use in serviceTable
func (r *Table) RefreshServiceByName(name string, key string) error {
	ori, ok := r.serviceTable.Load(ServiceNameString(name))
	if !ok {
		return r.CreateService(name, key)
	}
	return r.RefreshService(ori, key)
}

func (r *Table) RefreshService(ori *Service, key string) error {
	resp, err := utils.GetPrefixKV(r.cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return err
	}
	svr := &Service{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid service attribute key")
			return errors.NewFormat(200, "invalid service attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.NodeKeyString:
			var nodeSlice []string
			if err := json.Unmarshal(kv.Value, &nodeSlice); err != nil {
				logger.Error(err)
				return err
			}
			svr.ep = NewEndpointTableMap()
			for _, node := range nodeSlice {
				if ep, err := r.GetEndpointByName([]byte(node)); err != nil {
					// endpoint not exists in memory, maybe process it later
					logger.Debugf("endpoint [%s] not exists in memory, maybe process it later", node)
				} else {
					svr.ep.Store(ep.nameString, ep)
				}
			}
			confirm, _ := svr.checkEndpointStatus(Online)
			if len(confirm) > 0 {
				if err := svr.ResetOnlineEndpointRing(confirm); err != nil {
					logger.Error(err)
				}
			}
		case constant.NameKeyString:
			svr.name = kv.Value
			svr.nameString = ServiceNameString(kv.Value)
		default:
			logger.Errorf("unsupported service attribute: %s", keyStr)
			return errors.NewFormat(200, fmt.Sprintf("unsupported service attribute: %s", keyStr))
		}
	}
	ori.name = svr.name
	ori.nameString = svr.nameString
	ori.acceptHttpMethod = svr.acceptHttpMethod
	ori.ep = svr.ep
	ori.onlineEp = svr.onlineEp
	logger.Debugf("refresh service: %s", ori.nameString)

	r.routerTable.Range(func(key RouterNameString, value *Router) {
		if value.service.nameString == ori.nameString {
			// service connected
			value.service = ori
			if ok := value.CheckStatus(Online); ok {
				if _, err := r.SetRouterOnline(value); err != nil {
					return
				}
				if _, ok := r.onlineTable.Load(value.frontendApi); !ok {
					r.onlineTable.Store(value.frontendApi, value)
				}
			} else {
				if _, err := r.SetRouterStatus(value, BreakDown); err != nil {
					logger.Error(err)
					return
				}
				if _, ok := r.onlineTable.Load(value.frontendApi); !ok {
					r.onlineTable.Delete(value.frontendApi)
				}
			}
			if err := r.RefreshRouter(value, fmt.Sprintf("/Router/Router-%s/", string(value.name))); err != nil {
				logger.Error(err)
			}
		}
	})
	return nil
}

func (r *Table) DeleteService(name string) error {
	ori, ok := r.serviceTable.Load(ServiceNameString(name))
	if !ok {
		return nil
	}
	r.serviceTable.Delete(ServiceNameString(name))
	r.routerTable.Range(func(key RouterNameString, value *Router) {
		if value.service == ori {
			// service connected
			logger.Debugf("current router service: %+v", value.service)
			value.service = nil
			r.onlineTable.Delete(value.frontendApi)
		}
	})
	return nil
}

func (r *Table) CreateEndpoint(id string, key string) error {
	resp, err := utils.GetPrefixKV(r.cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return err
	}
	ep := &Endpoint{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid endpoint attribute key")
			return errors.NewFormat(200, "invalid endpoint attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.IdKeyString:
			// is forbidden to modify node id
			if id != string(kv.Value) {
				logger.Warnf("node key is not in accord with node id, node: %s", key)
				ep.id = id
			} else {
				ep.id = string(kv.Value)
			}
		case constant.NameKeyString:
			ep.name = kv.Value
			ep.nameString = EndpointNameString(kv.Value)
		case constant.HostKeyString:
			ep.host = kv.Value
		case constant.PortKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				logger.Error("wrong type of endpoint port")
				return err
			}
			ep.port = int(tmp)
		case constant.FailedTimesKeyString:
			// do nothing
		case constant.StatusKeyString:
			// do nothing
		case constant.HealthCheckKeyString:
			if hc, err := CreateHealthCheck(r.cli, id, constant.HealthCheckPrefixDefinition+id+constant.Slash); err != nil {
				logger.Error(err)
				return err
			} else {
				ep.healthCheck = hc
			}
		default:
			logger.Errorf("unsupported service attribute: %s", keyStr)
			return errors.NewFormat(200, fmt.Sprintf("unsupported service attribute: %s", keyStr))
		}
	}
	if ep.healthCheck != nil {
		if ok, err := ep.healthCheck.Check(ep.host, ep.port); err != nil {
			ep.setStatus(Offline)
		} else if !ok {
			ep.setStatus(BreakDown)
		} else {
			ep.setStatus(Online)
		}
	}
	r.endpointTable.Store(ep.nameString, ep)

	flag := false
	r.serviceTable.Range(func(key ServiceNameString, value *Service) bool {
		if ori, ok := value.ep.Load(ep.nameString); ok {
			ori.name = ep.name
			ori.nameString = ep.nameString
			ori.healthCheck = ep.healthCheck
			ori.port = ep.port
			ori.host = ep.host
			ori.id = ep.id
			ori.status = ep.status
			flag = true
			return true
		}
		return false
	})
	if !flag {
		resp, err := utils.GetPrefixKV(r.cli, constant.ServiceDefinition, clientv3.WithPrefix())
		if err != nil {
			logger.Error(err)
			return err
		}
		flag2 := false
		for _, kv := range resp.Kvs {
			tmp := bytes.TrimPrefix(kv.Key, []byte(constant.ServiceDefinition))
			svrSlice := bytes.Split(tmp, constant.SlashBytes)
			if len(svrSlice) < 2 {
				logger.Warnf("invalid endpoint attribute key")
				return errors.NewFormat(200, "invalid endpoint attribute key")
			}
			svr := svrSlice[0]
			key := svrSlice[1]
			svrName := bytes.TrimPrefix(svr, constant.ServicePrefixBytes)
			if string(key) == constant.NodeKeyString {
				var nodeSlice []string
				if err := json.Unmarshal(kv.Value, &nodeSlice); err != nil {
					logger.Error(err)
					return err
				}
				for _, n := range nodeSlice {
					if n == ep.id {
						// unprocessed endpoint found
						if s, err := r.GetServiceByName(svrName); err != nil {
							// service not found, do nothing
							// waiting create_service func to connect this endpoint
						} else {
							if err := r.RefreshService(s, fmt.Sprintf("/Service/Service-%s/", s.nameString)); err != nil {
								logger.Error(err)
							}
						}
						flag2 = true
						break
					}
				}
			}
		}
		if !flag2 {
			// this endpoint is not used by any service
			logger.Warnf("this endpoint is not used by any service")
		}
	}
	return nil
}

//not use in endpointTable
func (r *Table) RefreshEndpointById(id string, key string) error {
	oriEp, exist := r.GetEndpointById(id)
	if !exist {
		return r.CreateEndpoint(id, key)
	}
	return r.RefreshEndpoint(oriEp, key)
}

func (r *Table) RefreshEndpoint(oriEp *Endpoint, key string) error {
	var newStatus Status
	resp, err := utils.GetPrefixKV(r.cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return err
	}
	ep := &Endpoint{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid endpoint attribute key")
			return errors.NewFormat(200, "invalid endpoint attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.IdKeyString:
			// is forbidden to modify node id
			if oriEp.id != string(kv.Value) {
				logger.Warnf("node key is not in accord with node id, node: %s", key)
				ep.id = oriEp.id
			} else {
				ep.id = string(kv.Value)
			}
		case constant.NameKeyString:
			ep.name = kv.Value
			ep.nameString = EndpointNameString(kv.Value)
		case constant.HostKeyString:
			ep.host = kv.Value
		case constant.PortKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				logger.Error("wrong type of endpoint port")
				return err
			}
			ep.port = int(tmp)
		case constant.FailedTimesKeyString:
			// do nothing
		case constant.StatusKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return err
			}
			newStatus = Status(tmp)
		case constant.HealthCheckKeyString:
			if hc, err := RefreshHealthCheck(r.cli, oriEp.id, constant.HealthCheckPrefixDefinition+oriEp.id+constant.Slash); err != nil {
				logger.Error(err)
				return err
			} else {
				ep.healthCheck = hc
			}
		default:
			logger.Errorf("unsupported service attribute: %s", keyStr)
			return errors.NewFormat(200, fmt.Sprintf("unsupported service attribute: %s", keyStr))
		}
	}
	if ep.healthCheck != nil {
		if ok, err := ep.healthCheck.Check(ep.host, ep.port); err != nil || !ok {
			if newStatus == BreakDown {
				ep.setStatus(BreakDown)
			} else {
				ep.setStatus(Offline)
			}
		} else {
			ep.setStatus(Online)
		}
		if oriEp.status != ep.status {
			if err := r.SetEndpointStatus(oriEp, ep.status); err != nil {
				logger.Error(err)
			}
		}
		oriEp.healthCheck = ep.healthCheck
		oriEp.port = ep.port
		oriEp.host = ep.host
		oriEp.rate = ep.rate
	}

	r.serviceTable.Range(func(key ServiceNameString, value *Service) bool {
		if ori, ok := value.ep.Load(ep.nameString); ok {
			ori.name = ep.name
			ori.nameString = ep.nameString
			ori.healthCheck = ep.healthCheck
			ori.port = ep.port
			ori.host = ep.host
			ori.id = ep.id
			ori.status = ep.status

			if err := r.RefreshService(value, fmt.Sprintf("/Service/Service-%s/", value.nameString)); err != nil {
				logger.Error(err)
			}
		}
		return false
	})
	return nil
}

func (r *Table) DeleteEndpoint(id string) error {
	var err error
	if ep, exist := r.GetEndpointById(id); exist {
		logger.Debugf("endpoint delete begin: %s", id)
		r.endpointTable.Delete(ep.nameString)
		r.serviceTable.Range(func(key ServiceNameString, value *Service) bool {
			if _, ok := value.ep.Load(ep.nameString); ok {
				value.ep.Delete(ep.nameString)
				if err = r.RefreshService(value, fmt.Sprintf("/Service/Service-%s/", value.nameString)); err != nil {
					logger.Error(err)
					return true
				}
			}
			return false
		})
		logger.Debugf("endpoint delete finished: %s", id)
	}
	return err
}

func (r *Table) RefreshHealthCheck(id string, key string) error {
	var err error
	hc, err := RefreshHealthCheck(r.cli, id, key)
	if err != nil {
		logger.Error(err)
		return err
	}
	r.endpointTable.Range(func(key EndpointNameString, value *Endpoint) bool {
		if value.healthCheck.id == hc.id {
			value.healthCheck.path = hc.path
			value.healthCheck.retry = hc.retry
			value.healthCheck.retryTime = hc.retryTime
			value.healthCheck.interval = hc.interval
			value.healthCheck.timeout = hc.timeout

			if err = r.RefreshEndpoint(value, fmt.Sprintf("/Node/Node-%s/", value.id)); err != nil {
				logger.Error(err)
			}
			return true
		}
		return false
	})
	return err
}

func CreateHealthCheck(cli *clientv3.Client, id string, key string) (*HealthCheck, error) {
	resp, err := utils.GetPrefixKV(cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	hc := &HealthCheck{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid health-check attribute key")
			return nil, errors.NewFormat(200, "invalid health-check attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.IdKeyString:
			// is forbidden to modify health-check id
			if id != string(kv.Value) {
				logger.Warnf("node key is not in accord with node id, node: %s", key)
				hc.id = id
			} else {
				hc.id = string(kv.Value)
			}
		case constant.PathKeyString:
			hc.path = kv.Value
		case constant.IntervalKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			hc.interval = uint8(tmp)
		case constant.RetryKeyString:
			if string(kv.Value) == "0" {
				hc.retry = false
			} else if string(kv.Value) == "1" {
				hc.retry = true
			} else {
				logger.Warnf("unrecognized retry value: %+v", string(kv.Value))
				hc.retry = false
			}
		case constant.RetryTimeKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			hc.retryTime = uint8(tmp)
		case constant.TimeoutKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			hc.timeout = uint8(tmp)
		default:
			logger.Errorf("unsupported health-check attribute: %s", keyStr)
			return nil, errors.NewFormat(200, fmt.Sprintf("unsupported health-check attribute: %s", keyStr))
		}
	}
	return hc, nil
}

func RefreshHealthCheck(cli *clientv3.Client, id string, key string) (*HealthCheck, error) {
	resp, err := utils.GetPrefixKV(cli, key, clientv3.WithPrefix())
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	hc := &HealthCheck{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(key))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warnf("invalid health-check attribute key")
			return nil, errors.NewFormat(200, "invalid health-check attribute key")
		}
		keyStr := string(key)
		switch keyStr {
		case constant.IdKeyString:
			// is forbidden to modify health-check id
			if id != string(kv.Value) {
				logger.Warnf("node key is not in accord with node id, node: %s", key)
				hc.id = id
			} else {
				hc.id = string(kv.Value)
			}
		case constant.PathKeyString:
			hc.path = kv.Value
		case constant.IntervalKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			hc.interval = uint8(tmp)
		case constant.RetryKeyString:
			if string(kv.Value) == "0" {
				hc.retry = false
			} else if string(kv.Value) == "1" {
				hc.retry = true
			} else {
				logger.Warnf("unrecognized retry value: %+v", string(kv.Value))
				hc.retry = false
			}
		case constant.RetryTimeKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			hc.retryTime = uint8(tmp)
		case constant.TimeoutKeyString:
			tmp, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			hc.timeout = uint8(tmp)
		default:
			logger.Errorf("unsupported health-check attribute: %s", keyStr)
			return nil, errors.NewFormat(200, fmt.Sprintf("unsupported health-check attribute: %s", keyStr))
		}
	}
	return hc, nil
}
