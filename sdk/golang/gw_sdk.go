package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
	"github.com/deckarep/golang-set"
	"os"
	"strconv"
	"time"
)

type Node struct {
	ID     string
	Name   string
	Host   string
	Port   int
	Status uint8
	HC     *HealthCheck
}
type Service struct {
	Name string
	Node []*Node
}
type Router struct {
	ID       string
	Name     string
	Status   uint8
	Frontend string
	Backend  string
	Service  *Service
}
type HealthCheck struct {
	ID        string
	Path      string
	Timeout   uint8
	Interval  uint8
	Retry     bool
	RetryTime uint8
}

type ApiGatewayRegistrant struct {
	cli *clientv3.Client

	node    *Node
	service *Service
	hc      *HealthCheck
	router  []*Router
}

var (
	sdkLogger = log.New()
)

func NewHealthCheck(path string, timeout, interval, retryTime uint8, retry bool) *HealthCheck {
	var uid string

	hardwareAddr := GetHardwareAddressAsLong()
	if len(hardwareAddr) > 0 {
		// use hardware address of first interface
		uid = strconv.FormatInt(hardwareAddr[0], 10)
	} else {
		sdkLogger.Error("can not gain local hardware address")
	}

	return &HealthCheck{
		ID:        uid,
		Path:      path,
		Timeout:   timeout,
		Interval:  interval,
		Retry:     retry,
		RetryTime: retryTime,
	}
}

func NewNode(name, host string, port int, hc *HealthCheck) *Node {
	var uid string

	hardwareAddr := GetHardwareAddressAsLong()
	if len(hardwareAddr) > 0 {
		// use hardware address of first interface
		uid = strconv.FormatInt(hardwareAddr[0], 10)
	} else {
		sdkLogger.Error("can not gain local hardware address")
	}

	return &Node{
		ID:     uid,
		Name:   name + uid,
		Host:   host,
		Port:   port,
		Status: 0,
		HC:     hc,
	}
}

func NewService(name string, node *Node) *Service {
	return &Service{
		Name: name,
		Node: []*Node{node},
	}
}

func NewRouter(name, frontend, backend string, service *Service) *Router {
	var uid string

	hardwareAddr := GetHardwareAddressAsLong()
	if len(hardwareAddr) > 0 {
		// use hardware address of first interface
		uid = strconv.FormatInt(hardwareAddr[0], 10)
	} else {
		sdkLogger.Error("can not gain local hardware address")
	}

	return &Router{
		ID:       uid,
		Name:     name,
		Status:   0,
		Frontend: frontend,
		Backend:  backend,
		Service:  service,
	}
}

func NewApiGatewayRegistrant(cli *clientv3.Client, node *Node, service *Service, router []*Router) ApiGatewayRegistrant {
	return ApiGatewayRegistrant{
		cli:     cli,
		node:    node,
		service: service,
		hc:      node.HC,
		router:  router,
	}
}

func (gw *ApiGatewayRegistrant) getKeyValue(key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if gw.cli == nil {
		sdkLogger.Error("etcd client need initialize")
	}
	cli := gw.cli

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	return resp, err
}

func (gw *ApiGatewayRegistrant) getKeyValueWithPrefix(key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if gw.cli == nil {
		sdkLogger.Error("etcd client need initialize")
	}
	cli := gw.cli

	opts = append(opts, clientv3.WithPrefix())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	return resp, err
}

func (gw *ApiGatewayRegistrant) putKeyValue(key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	if gw.cli == nil {
		sdkLogger.Error("etcd client need initialize")
	}
	cli := gw.cli

	opts = append(opts, clientv3.WithPrefix())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Put(ctx, key, value, opts...)
	cancel()
	return resp, err
}

func (gw *ApiGatewayRegistrant) putMany(kv interface{}, opts ...clientv3.OpOption) error {
	if gw.cli == nil {
		sdkLogger.Error("etcd client need initialize")
	}
	cli := gw.cli

	kvs, ok := kv.(map[string]interface{})
	if !ok {
		return errors.New("wrong type of kv mapping")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	for k, v := range kvs {
		if sliceValue, ok := v.([]string); ok {
			for _, item := range sliceValue {
				_, err := cli.Put(ctx, k, item, opts...)
				if err != nil {
					cancel()
					return err
				}
			}
		} else if strValue, ok := v.(string); ok {
			_, err := cli.Put(ctx, k, strValue, opts...)
			if err != nil {
				cancel()
				return err
			}
		} else {
			cancel()
			return errors.New("wrong type of kv mapping value")
		}
	}
	cancel()
	return nil
}

func (gw *ApiGatewayRegistrant) registerNode() error {
	var kvs map[string]interface{}
	kvs = make(map[string]interface{})

	nodeDefinition := fmt.Sprintf(NodeDefinition, gw.node.ID)
	resp, err := gw.getKeyValueWithPrefix(nodeDefinition)
	if err != nil {
		sdkLogger.Exception(err)
		return err
	}
	n := gw.node
	if resp.Count == 0 {
		kvs[nodeDefinition+IDKey] = n.ID
		kvs[nodeDefinition+NameKey] = n.Name
		kvs[nodeDefinition+HostKey] = n.Host
		kvs[nodeDefinition+PortKey] = strconv.FormatInt(int64(n.Port), 10)
		kvs[nodeDefinition+StatusKey] = strconv.FormatUint(uint64(n.Status), 10)
		kvs[nodeDefinition+HealthCheckKey] = n.HC.ID
	} else {
		for _, kv := range resp.Kvs {
			if bytes.Equal(kv.Key, []byte(nodeDefinition+StatusKey)) {
				// pass
			} else if bytes.Equal(kv.Key, []byte(nodeDefinition + IDKey)) {
				if !bytes.Equal(kv.Value, []byte(n.ID)) {
					kvs[nodeDefinition+IDKey] = n.ID
				}
			} else if bytes.Equal(kv.Key, []byte(nodeDefinition + NameKey)) {
				if !bytes.Equal(kv.Value, []byte(n.Name)) {
					kvs[nodeDefinition+NameKey] = n.Name
				}
			} else if bytes.Equal(kv.Key, []byte(nodeDefinition + StatusKey)) {
				if !bytes.Equal(kv.Value, []byte(strconv.FormatUint(uint64(n.Status), 10))) {
					kvs[nodeDefinition+StatusKey] = strconv.FormatUint(uint64(n.Status), 10)
				}
			} else if bytes.Equal(kv.Key, []byte(nodeDefinition + HostKey)) {
				if !bytes.Equal(kv.Value, []byte(n.Host)) {
					kvs[nodeDefinition+HostKey] = n.Host
				}
			} else if bytes.Equal(kv.Key, []byte(nodeDefinition + PortKey)) {
				if !bytes.Equal(kv.Value, []byte(strconv.FormatInt(int64(n.Port), 10))) {
					kvs[nodeDefinition+BackendKey] = strconv.FormatInt(int64(n.Port), 10)
				}
			} else if bytes.Equal(kv.Key, []byte(nodeDefinition + HealthCheckKey)) {
				if !bytes.Equal(kv.Value, []byte(n.HC.ID)) {
					kvs[nodeDefinition+ServiceKey] = n.HC.ID
				}
			} else {
				sdkLogger.Warningf("unrecognized node key: %s", string(kv.Key))
			}
		}
		sdkLogger.Infof("node keys waiting to be updated: %+v", kvs)
	}

	err = gw.putMany(kvs)
	if err != nil {
		sdkLogger.Exception(err)
		return err
	}

	hc := gw.hc
	hcDefinition := fmt.Sprintf(HealthCheckDefinition, hc.ID)
	resp, err = gw.getKeyValueWithPrefix(hcDefinition)
	if err != nil {
		sdkLogger.Exception(err)
		return err
	}
	kvs = make(map[string]interface{})

	if resp.Count == 0 {
		kvs[hcDefinition+IDKey] = hc.ID
		kvs[hcDefinition+PathKey] = hc.Path
		kvs[hcDefinition+TimeoutKey] = strconv.FormatUint(uint64(hc.Timeout), 10)
		kvs[hcDefinition+IntervalKey] = strconv.FormatUint(uint64(hc.Interval), 10)
		if hc.Retry {
			kvs[hcDefinition+RetryKey] = "1"
		} else {
			kvs[hcDefinition+RetryKey] = "0"
		}
		kvs[hcDefinition+RetryTimeKey] = strconv.FormatUint(uint64(hc.RetryTime), 10)
	} else {
		for _, kv := range resp.Kvs {
			if bytes.Equal(kv.Key, []byte(hcDefinition+StatusKey)) {
				// pass
			} else if bytes.Equal(kv.Key, []byte(hcDefinition + IDKey)) {
				if !bytes.Equal(kv.Value, []byte(hc.ID)) {
					kvs[hcDefinition+IDKey] = hc.ID
				}
			} else if bytes.Equal(kv.Key, []byte(hcDefinition + PathKey)) {
				if !bytes.Equal(kv.Value, []byte(hc.Path)) {
					kvs[hcDefinition+PathKey] = hc.Path
				}
			} else if bytes.Equal(kv.Key, []byte(hcDefinition + TimeoutKey)) {
				if !bytes.Equal(kv.Value, []byte(strconv.FormatUint(uint64(hc.Timeout), 10))) {
					kvs[hcDefinition+TimeoutKey] = strconv.FormatUint(uint64(hc.Timeout), 10)
				}
			} else if bytes.Equal(kv.Key, []byte(hcDefinition + IntervalKey)) {
				if !bytes.Equal(kv.Value, []byte(strconv.FormatUint(uint64(hc.Interval), 10))) {
					kvs[hcDefinition+IntervalKey] = strconv.FormatUint(uint64(hc.Interval), 10)
				}
			} else if bytes.Equal(kv.Key, []byte(hcDefinition + RetryKey)) {
				var retry string
				if hc.Retry {
					retry = "1"
				} else {
					retry = "0"
				}
				if !bytes.Equal(kv.Value, []byte(retry)) {
					kvs[hcDefinition+RetryKey] = retry
				}
			} else if bytes.Equal(kv.Key, []byte(hcDefinition + RetryTimeKey)) {
				if !bytes.Equal(kv.Value, []byte(strconv.FormatUint(uint64(hc.RetryTime), 10))) {
					kvs[hcDefinition+RetryTimeKey] = strconv.FormatUint(uint64(hc.RetryTime), 10)
				}
			} else {
				sdkLogger.Warningf("unrecognized node key: %s", string(kv.Key))
			}
		}
		sdkLogger.Infof("node keys waiting to be updated: %+v", kvs)
	}

	err = gw.putMany(kvs)
	if err != nil {
		sdkLogger.Exception(err)
		return err
	}

	return nil
}

func (gw *ApiGatewayRegistrant) registerService() error {
	var kvs map[string]interface{}
	kvs = make(map[string]interface{})

	serviceDefinition := fmt.Sprintf(ServiceDefinition, gw.service.Name)
	resp, err := gw.getKeyValueWithPrefix(serviceDefinition)
	if err != nil {
		sdkLogger.Exception(err)
		return err
	}
	if resp.Count == 0 {
		// service not existed
		var nodes []string
		for _, n := range gw.service.Node {
			nodes = append(nodes, n.ID)
		}
		nodeSlice, err := json.Marshal(nodes)
		if err != nil {
			sdkLogger.Exception(err)
			return err
		}

		kvs[serviceDefinition+NameKey] = gw.service.Name
		kvs[serviceDefinition+NodeKey] = string(nodeSlice)
	} else {
		resp, err := gw.getKeyValue(serviceDefinition + NodeKey)
		if err != nil {
			sdkLogger.Exception(err)
			return err
		}
		if resp.Count == 0 {
			// a ri e na i, impossible!
			return errors.New(fmt.Sprintf("can not find key: [%s]", serviceDefinition+NodeKey))
		} else if resp.Count > 0 {
			var nodeSlice []string
			err := json.Unmarshal(resp.Kvs[0].Value, &nodeSlice)
			if err != nil {
				sdkLogger.Exception(err)
				return err
			}
			s := mapset.NewSet()
			for _, node := range nodeSlice {
				s.Add(node)
			}
			if !s.Contains(gw.node.ID) {
				s.Add(gw.node.ID)
				nodeByteSlice, err := json.Marshal(s.ToSlice())
				if err != nil {
					sdkLogger.Exception(err)
					os.Exit(-1)
				} else {
					kvs[serviceDefinition+NodeKey] = string(nodeByteSlice)
				}
			}
		}
	}
	err = gw.putMany(kvs)
	if err != nil {
		sdkLogger.Exception(err)
		return err
	}
	return nil
}

func (gw *ApiGatewayRegistrant) registerRouter() error {
	for _, r := range gw.router {
		var kvs map[string]interface{}
		kvs = make(map[string]interface{})

		routerName := fmt.Sprintf(RouterDefinition, r.Name)
		resp, err := gw.getKeyValueWithPrefix(routerName)
		if err != nil {
			sdkLogger.Exception(err)
			return err
		}
		if resp.Count == 0 {
			kvs[routerName+IDKey] = r.ID
			kvs[routerName+NameKey] = r.Name
			kvs[routerName+StatusKey] = strconv.FormatUint(uint64(r.Status), 10)
			kvs[routerName+FrontendKey] = r.Frontend
			kvs[routerName+BackendKey] = r.Backend
			kvs[routerName+ServiceKey] = r.Service.Name
		} else {
			for _, kv := range resp.Kvs {
				if bytes.Equal(kv.Key, []byte(routerName+StatusKey)) {
					if string(kv.Value) == "1" {
						// router still online, can not be updated
						sdkLogger.Info("router still online, can not be updated")
						break
					}
				} else if bytes.Equal(kv.Key, []byte(routerName + IDKey)) {
					if !bytes.Equal(kv.Value, []byte(r.ID)) {
						kvs[routerName+IDKey] = r.ID
					}
				} else if bytes.Equal(kv.Key, []byte(routerName + NameKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Name)) {
						kvs[routerName+NameKey] = r.Name
					}
				} else if bytes.Equal(kv.Key, []byte(routerName + StatusKey)) {
					if !bytes.Equal(kv.Value, []byte(strconv.FormatUint(uint64(r.Status), 10))) {
						kvs[routerName+StatusKey] = strconv.FormatUint(uint64(r.Status), 10)
					}
				} else if bytes.Equal(kv.Key, []byte(routerName + FrontendKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Frontend)) {
						kvs[routerName+FrontendKey] = r.Frontend
					}
				} else if bytes.Equal(kv.Key, []byte(routerName + BackendKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Backend)) {
						kvs[routerName+BackendKey] = r.Backend
					}
				} else if bytes.Equal(kv.Key, []byte(routerName + ServiceKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Service.Name)) {
						kvs[routerName+ServiceKey] = r.Service.Name
					}
				} else {
					sdkLogger.Warningf("unrecognized router key: %s", string(kv.Key))
				}
			}
			sdkLogger.Infof("router keys waiting to be updated: %+v", kvs)
		}
		err = gw.putMany(kvs)
		if err != nil {
			sdkLogger.Exception(err)
			return err
		}
	}

	return nil
}

func (gw *ApiGatewayRegistrant) Register() error {
	fmt.Print(gw.registerNode())
	fmt.Print(gw.registerService())
	fmt.Print(gw.registerRouter())
	return nil
}
