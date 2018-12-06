package core

import (
	"bytes"
	"context"
	"encoding/json"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"os"
	"strconv"
	"time"
)

const (
	Root                        = "/"
	Slash                       = "/"
	RouterDefinition            = "/Router/"
	RouterPrefixString          = "Router-%s/"
	ServiceDefinition           = "/Service/"
	NodeDefinition              = "/Node/"
	NodePrefixString            = "Node-%s/"
	NodePrefixDefinition        = "/Node/Node-"
	HealthCheckPrefixDefinition = "/HealthCheck/HC-"
	StatusKeyString             = "Status"
	FailedTimesKeyString = "FailedTimes"
)

var (
	SlashBytes             = []byte("/")
	NodeKeyBytes           = []byte("Node")
	ServiceKeyBytes        = []byte("Service")
	ServicePrefixBytes     = []byte("Service-")
	RouterPrefixBytes      = []byte("Router-")
	HealthCheckKeyBytes    = []byte("HealthCheck")
	IdKeyBytes             = []byte("ID")
	NameKeyBytes           = []byte("Name")
	HostKeyBytes           = []byte("Host")
	PortKeyBytes           = []byte("Port")
	StatusKeyBytes         = []byte("Status")
	FrontendApiKeyBytes    = []byte("FrontendApi")
	BackendApiKeyBytes     = []byte("BackendApi")
	PathKeyBytes           = []byte("Path")
	TimeoutKeyBytes        = []byte("Timeout")
	IntervalKeyBytes       = []byte("Interval")
	RetryKeyBytes          = []byte("Retry")
	RetryTimeKeyBytes      = []byte("RetryTime")
	RouterDefinitionBytes  = []byte("/Router/")
	ServiceDefinitionBytes = []byte("/Service/")
)

func RouterWatcher(watchChannel clientv3.WatchChan) {
	for {
		resp := <-watchChannel
		for _, i := range resp.Events {

			logger.Info(i)
		}
	}
}

func InitRoutingTable(cli *clientv3.Client) *RoutingTable {
	var rt RoutingTable
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
				rt.SetEndpointOnline(ep)
				ep.setStatus(Online)
			} else {
				rt.SetEndpointStatus(ep, BreakDown)
				ep.setStatus(BreakDown)
				logger.Errorf("Endpoint {%s:%d} health check failed: %s", string(ep.host), ep.port, err.Error())
			}
		}
	}
	rt.routerTable.Range(func(key RouterNameString, value *Router) {
		if value.CheckStatus(Online) {
			rt.SetRouterStatus(value, Online)
		} else {
			rt.SetRouterStatus(value, Offline)
		}
		confirm, _ := value.service.checkEndpointStatus(Online)
		if err := value.service.ResetOnlineEndpointRing(confirm); err != nil {
			logger.Error(err.(errors.Error).String())
		}
	})
	return &rt
}

func initServiceNode(cli *clientv3.Client) (*ServiceTableMap, *EndpointTableMap, error) {
	svrMap := NewServiceTableMap()
	epMap := NewEndpointTableMap()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, ServiceDefinition, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, ServiceDefinitionBytes)
		tmp := bytes.Split(key, SlashBytes)
		if len(tmp) != 2 {
			logger.Infof("invalid service definition: %s", key)
			continue
		} else {
			sName := bytes.TrimPrefix(tmp[0], ServicePrefixBytes)
			if s, ok := svrMap.Load(ServiceNameString(sName)); ok {
				// service exists
				if bytes.Equal(tmp[1], NameKeyBytes) {
					s.name = kv.Value
					s.nameString = ServiceNameString(kv.Value)
				} else if bytes.Equal(tmp[1], NodeKeyBytes) {
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
							logger.Exception(err)
							return nil, nil, err
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
					logger.Warningf("unrecognized node attribute\n\tkey: %s\n\tvalue: %s", string(kv.Key), string(kv.Value))
				}
			} else {
				if bytes.Equal(tmp[1], NameKeyBytes) {
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
				} else if bytes.Equal(tmp[1], NodeKeyBytes) {
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
							logger.Exception(err)
							return nil, nil, err
						}
						epSlice.Store(ep.nameString, ep)
					}
					s.ep = epSlice
					svrMap.Store(s.nameString, s)
					s.ep.Range(func(key EndpointNameString, value *Endpoint) {
						epMap.Store(key, value)
					})
				} else {
					logger.Warningf("unrecognized node attribute\n\tkey: %s\n\tvalue: %s", string(kv.Key), string(kv.Value))
				}
			}
		}
	}
	return svrMap, epMap, nil
}

func initEndpointNode(cli *clientv3.Client, nodeID string) (*Endpoint, error) {
	var ep Endpoint

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, NodePrefixDefinition+nodeID, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
		return nil, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(NodePrefixDefinition+nodeID+Slash))
		if bytes.Equal(key, IdKeyBytes) {
			ep.id = string(kv.Value)
		} else if bytes.Equal(key, NameKeyBytes) {
			ep.name = kv.Value
			ep.nameString = EndpointNameString(kv.Value)
		} else if bytes.Equal(key, HostKeyBytes) {
			ep.host = kv.Value
		} else if bytes.Equal(key, PortKeyBytes) {
			tmpInt, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				logger.Exception(err)
				return nil, err
			}
			ep.port = int(tmpInt)
		} else if bytes.Equal(key, StatusKeyBytes) {
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
		} else if bytes.Equal(key, HealthCheckKeyBytes) {
			// get health check info
			ctxA, cancelA := context.WithTimeout(context.Background(), 1*time.Second)
			respA, err := cli.Get(ctxA, HealthCheckPrefixDefinition+string(kv.Value), clientv3.WithPrefix())
			cancelA()
			if err != nil {
				logger.Exception(err)
				return nil, err
			}

			var hc HealthCheck
			hc.path = nil
			for _, kvA := range respA.Kvs {
				keyA := bytes.TrimPrefix(kvA.Key, []byte(HealthCheckPrefixDefinition+string(kv.Value)+Slash))
				if bytes.Equal(keyA, IdKeyBytes) {
					hc.id = string(kvA.Value)
				} else if bytes.Equal(keyA, PathKeyBytes) {
					hc.path = kvA.Value
				} else if bytes.Equal(keyA, TimeoutKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Exception(err)
						return nil, err
					}
					hc.timeout = uint8(tmpInt)
				} else if bytes.Equal(keyA, IntervalKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Exception(err)
						return nil, err
					}
					hc.interval = uint8(tmpInt)
				} else if bytes.Equal(keyA, RetryKeyBytes) {
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
				} else if bytes.Equal(keyA, RetryTimeKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10, 64)
					if err != nil {
						logger.Exception(err)
						return nil, err
					}
					hc.retryTime = uint8(tmpInt)
				} else {
					// unrecognized attribute
					logger.Warningf("unrecognized health check attribute\n\tkey: %s\n\tvalue: %s", string(kvA.Key), string(kvA.Value))
				}
			}
			ep.healthCheck = &hc
		} else {
			logger.Warningf("unrecognized node attribute\n\tkey: %s\n\tvalue: %s", string(kv.Key), string(kv.Value))
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
	resp, err := cli.Get(ctx, RouterDefinition, clientv3.WithPrefix())
	cancel()
	if err != nil {
		logger.Exception(err)
		return nil, nil, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, RouterDefinitionBytes)
		tmpSlice := bytes.Split(key, SlashBytes)
		if len(tmpSlice) != 2 {
			logger.Warningf("invalid router definition: %s", key)
			continue
		} else {
			rName := bytes.TrimPrefix(tmpSlice[0], RouterPrefixBytes)
			attr := tmpSlice[1]
			if r, ok := rtMap.Load(RouterNameString(rName)); ok {
				if bytes.Equal(attr, IdKeyBytes) {
					// do nothing
				} else if bytes.Equal(attr, NameKeyBytes) {
					if !bytes.Equal(rName, kv.Value) {
						logger.Warningf("inconsistent router definition: %s %s", string(kv.Key), string(kv.Value))
						continue
					}
					r.name = kv.Value
				} else if bytes.Equal(attr, FrontendApiKeyBytes) {
					r.frontendApi = &FrontendApi{
						path:       kv.Value,
						pathString: FrontendApiString(kv.Value),
						pattern:    bytes.Split(kv.Value, SlashBytes),
					}
				} else if bytes.Equal(attr, BackendApiKeyBytes) {
					r.backendApi = &BackendApi{
						path:       kv.Value,
						pathString: BackendApiString(kv.Value),
						pattern:    bytes.Split(kv.Value, SlashBytes),
					}
				} else if bytes.Equal(attr, ServiceKeyBytes) {
					if svr, ok := svrMap.Load(ServiceNameString(kv.Value)); ok {
						r.service = svr
					} else {
						logger.Errorf("service not exist")
						continue
					}
				} else if bytes.Equal(attr, StatusKeyBytes) {
					// do nothing
				} else {
					logger.Warningf("unrecognized health check attribute\n\tkey: %s\n\tvalue: %s", string(kv.Key), string(kv.Value))
				}
			} else {
				tmpRouter := Router{}
				if bytes.Equal(attr, IdKeyBytes) {
					// do nothing
				} else if bytes.Equal(attr, NameKeyBytes) {
					if !bytes.Equal(rName, kv.Value) {
						logger.Warningf("inconsistent router definition: %s %s", string(kv.Key), string(kv.Value))
						continue
					} else {
						tmpRouter.name = rName
					}
					rtMap.Store(RouterNameString(rName), &tmpRouter)
				} else if bytes.Equal(attr, FrontendApiKeyBytes) {
					tmpRouter.frontendApi = &FrontendApi{
						path:       kv.Value,
						pathString: FrontendApiString(kv.Value),
						pattern:    bytes.Split(kv.Value, SlashBytes),
					}
					rtMap.Store(RouterNameString(rName), &tmpRouter)
				} else if bytes.Equal(attr, BackendApiKeyBytes) {
					tmpRouter.backendApi = &BackendApi{
						path:       kv.Value,
						pathString: BackendApiString(kv.Value),
						pattern:    bytes.Split(kv.Value, SlashBytes),
					}
					rtMap.Store(RouterNameString(rName), &tmpRouter)
				} else if bytes.Equal(attr, ServiceKeyBytes) {
					if svr, ok := svrMap.Load(ServiceNameString(kv.Value)); ok {
						tmpRouter.service = svr
						rtMap.Store(RouterNameString(rName), &tmpRouter)
					} else {
						logger.Error("service not exist")
						continue
					}
				} else {
					logger.Warningf("unrecognized health check attribute\n\tkey: %s\n\tvalue: %s", string(kv.Key), string(kv.Value))
				}
			}
		}
	}

	rtMap.Range(func(key RouterNameString, value *Router) {
		artMap.Store(value.frontendApi.pathString, value)
	})
	return rtMap, artMap, nil
}
