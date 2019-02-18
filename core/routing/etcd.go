package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"os"
	"strconv"
	"time"
)

func InitRoutingTable(cli *clientv3.Client) *Table {
	var rt Table
	var epSlice []*Endpoint

	rt.cli = cli
	rt.Version = "1.0.0"
	ol := NewOnlineRouteTableMap()
	svrMap, epMap, err := initServiceNode(cli)
	if err != nil {
		logger.Exception(err)
		os.Exit(-1)
	}
	rt.serviceTable = *svrMap
	rt.endpointTable = *epMap
	rt.onlineTable = *ol
	routerTable, table, err := initRouter(cli, &rt.serviceTable)
	if err != nil {
		logger.Exception(err)
	}
	rt.table = *table
	rt.routerTable = *routerTable

	rt.endpointTable.Range(func(key EndpointNameString, value *Endpoint) {
		if value.healthCheck.path != nil {
			epSlice = append(epSlice, value)
		}
	})
	if len(epSlice) > 0 {
		for _, ep := range epSlice {
			logger.Debugf("ep {name: %s} {host: %s} {port: %d} {status: %d}", ep.nameString, string(ep.host), ep.port, ep.status)
			if check, err := ep.healthCheck.Check(ep.host, ep.port); check {
				if err = rt.SetEndpointOnline(ep); err != nil {
					ep.setStatus(Online)
				}
			} else {
				if err = rt.SetEndpointStatus(ep, BreakDown); err != nil {
					ep.setStatus(BreakDown)
				}
				logger.Errorf("Endpoint {%s:%d} health check failed: %s", string(ep.host), ep.port, err.Error())
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
				logger.Error(err.(errors.Error).String())
			}
		}
	})
	return &rt
}

func initServiceNode(cli *clientv3.Client) (*ServiceTableMap, *EndpointTableMap, error) {
	svrMap := NewServiceTableMap()
	epMap := NewEndpointTableMap()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, constant.ServiceDefinition, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
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
						logger.Exception(err)
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
					s.ep.Range(func(key EndpointNameString, value *Endpoint) {
						epMap.Store(key, value)
					})
				} else {
					logger.Warningf("unrecognized node attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
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
						logger.Exception(err)
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
					s.ep.Range(func(key EndpointNameString, value *Endpoint) {
						epMap.Store(key, value)
					})
				} else {
					logger.Warningf("unrecognized node attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
				}
			}
		}
	}
	return svrMap, epMap, nil
}

func initEndpointNode(cli *clientv3.Client, nodeID string) (*Endpoint, error) {
	var ep Endpoint

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, constant.NodePrefixDefinition+nodeID, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
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
				logger.Exception(err)
				return nil, err
			}
			ep.port = int(tmpInt)
		} else if bytes.Equal(key, constant.StatusKeyBytes) {
			tmpInt, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				logger.Exception(err)
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
		} else if bytes.Equal(key, constant.HealthCheckKeyBytes) {
			// get health check info
			ctxA, cancelA := context.WithTimeout(context.Background(), 1*time.Second)
			respA, err := cli.Get(ctxA, constant.HealthCheckPrefixDefinition+string(kv.Value), clientv3.WithPrefix())
			cancelA()
			if err != nil {
				logger.Exception(err)
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
						logger.Exception(err)
						return nil, err
					}
					hc.timeout = uint8(tmpInt)
				} else if bytes.Equal(keyA, constant.IntervalKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Exception(err)
						return nil, err
					}
					hc.interval = uint8(tmpInt)
				} else if bytes.Equal(keyA, constant.RetryKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Exception(err)
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
						logger.Exception(err)
						return nil, err
					}
					hc.retryTime = uint8(tmpInt)
				} else {
					// unrecognized attribute
					logger.Warningf("unrecognized health check attribute, key: %s, value: %s", string(kvA.Key), string(kvA.Value))
				}
			}
			ep.healthCheck = &hc
		} else {
			logger.Warningf("unrecognized node attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
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

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, constant.RouterDefinition, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
		return nil, nil, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, constant.RouterDefinitionBytes)
		tmpSlice := bytes.Split(key, constant.SlashBytes)
		if len(tmpSlice) != 2 {
			logger.Warningf("invalid router definition: %s", key)
			continue
		} else {
			rName := bytes.TrimPrefix(tmpSlice[0], constant.RouterPrefixBytes)
			attr := tmpSlice[1]
			if r, ok := rtMap.Load(RouterNameString(rName)); ok {
				if bytes.Equal(attr, constant.IdKeyBytes) {
					// do nothing
				} else if bytes.Equal(attr, constant.NameKeyBytes) {
					if !bytes.Equal(rName, kv.Value) {
						logger.Warningf("inconsistent router definition: %s %s", string(kv.Key), string(kv.Value))
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
					logger.Warningf("unrecognized health check attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
				}
			} else {
				tmpRouter := Router{}
				if bytes.Equal(attr, constant.IdKeyBytes) {
					// do nothing
				} else if bytes.Equal(attr, constant.NameKeyBytes) {
					if !bytes.Equal(rName, kv.Value) {
						logger.Warningf("inconsistent router definition: %s %s", string(kv.Key), string(kv.Value))
						continue
					} else {
						tmpRouter.name = rName
					}
					rtMap.Store(RouterNameString(rName), &tmpRouter)
				} else if bytes.Equal(attr, constant.FrontendApiKeyBytes) {
					tmpRouter.frontendApi = &FrontendApi{
						path:       kv.Value,
						pathString: FrontendApiString(kv.Value),
						pattern:    bytes.Split(kv.Value, constant.SlashBytes),
					}
					rtMap.Store(RouterNameString(rName), &tmpRouter)
				} else if bytes.Equal(attr, constant.BackendApiKeyBytes) {
					tmpRouter.backendApi = &BackendApi{
						path:       kv.Value,
						pathString: BackendApiString(kv.Value),
						pattern:    bytes.Split(kv.Value, constant.SlashBytes),
					}
					rtMap.Store(RouterNameString(rName), &tmpRouter)
				} else if bytes.Equal(attr, constant.ServiceKeyBytes) {
					if svr, ok := svrMap.Load(ServiceNameString(kv.Value)); ok {
						tmpRouter.service = svr
						rtMap.Store(RouterNameString(rName), &tmpRouter)
					} else {
						logger.Error("service not exist")
						continue
					}
				} else {
					logger.Warningf("unrecognized health check attribute, key: %s, value: %s", string(kv.Key), string(kv.Value))
				}
			}
		}
	}

	rtMap.Range(func(key RouterNameString, value *Router) {
		artMap.Store(value.frontendApi.pathString, value)
	})
	return rtMap, artMap, nil
}

func (r *Table) CreateRouter(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := r.cli.Get(ctx, name, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
		return err
	}
	router := &Router{}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(constant.RouterDefinition + fmt.Sprintf(constant.RouterPrefixString, name)))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warningf("invalid router attribute key")
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
	r.table.Store(router.frontendApi.pathString, router)
	r.routerTable.Store(RouterNameString(router.name), router)
	confirm, _ := router.service.checkEndpointStatus(Online)
	if len(confirm) > 0 {
		if err := router.service.ResetOnlineEndpointRing(confirm); err != nil {
			logger.Exception(err)
			return err
		}
		if _, err := r.SetRouterOnline(router); err != nil {
			logger.Exception(err)
			return err
		}
	}
	return nil
}

func (r *Table) RefreshRouter(name string) error {
	router, err := r.GetRouterByName([]byte(name))
	if err != nil {
		// can not find router in routing table, try to generate a new one
		return r.CreateRouter(name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := r.cli.Get(ctx, name, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
		return err
	}
	if router.CheckStatus(Online) {
		// if router is online now, it will not be refreshed
		// TODO: trigger alarm

		return errors.New(132)
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(constant.RouterDefinition + fmt.Sprintf(constant.RouterPrefixString, name)))
		if bytes.Contains(key, constant.SlashBytes) {
			logger.Warningf("invalid router attribute key")
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
			logger.Exception(err)
			return err
		}
		if _, err := r.SetRouterOnline(router); err != nil {
			logger.Exception(err)
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
	return nil
}