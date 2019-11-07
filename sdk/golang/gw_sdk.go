package golang

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
	"github.com/deckarep/golang-set"
	"os"
	"strconv"
	"strings"
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
	Method   string
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
	logger = log.Logger
)

func NewHealthCheck(path string, timeout, interval, retryTime uint8, retry bool) *HealthCheck {
	return &HealthCheck{
		Path:      path,
		Timeout:   timeout,
		Interval:  interval,
		Retry:     retry,
		RetryTime: retryTime,
	}
}

func NewNode(host string, port int, hc *HealthCheck) *Node {
	var uid string

	hardwareAddr := GetHardwareAddressAsLong()
	if len(hardwareAddr) > 0 {
		// use hardware address of first interface
		uid = strconv.FormatInt(hardwareAddr[0], 10) + ":" + strconv.FormatInt(int64(port), 10)
	} else {
		logger.Error("can not gain local hardware address")
	}
	hc.ID = uid
	return &Node{
		ID:     uid,
		Name:   uid,
		Host:   host,
		Port:   port,
		Status: 2,
		HC:     hc,
	}
}

func NewService(name string, node *Node) *Service {
	return &Service{
		Name: name,
		Node: []*Node{node},
	}
}

func NewRouter(name, method, frontend, backend string, service *Service) *Router {
	method = strings.ToUpper(method)
	src := fmt.Sprintf("%s - %s - %s - %s - %s", name, method, frontend, backend, service.Name)
	fmt.Println(">> GATE route: ", src)
	if strings.Contains(name, "/") {
		logger.Errorf("name can not contains '/'")
		os.Exit(-1)
	}
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(src)))
	return &Router{
		ID:       md5str,
		Name:     name,
		Status:   0,
		Method:   method,
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
		logger.Error("etcd client need initialize")
		return nil, errors.New("etcd client need initialize")
	}
	cli := gw.cli

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	return resp, err
}

func (gw *ApiGatewayRegistrant) getKeyValueWithPrefix(key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if gw.cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.New("etcd client need initialize")
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
		logger.Error("etcd client need initialize")
		return nil, errors.New("etcd client need initialize")
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
		logger.Error("etcd client need initialize")
		return errors.New("etcd client need initialize")
	}
	cli := gw.cli

	kvs, ok := kv.(map[string]interface{})
	if !ok {
		logger.Error("wrong type of kv mapping")
		return errors.New("wrong type of kv mapping")
	}
	if len(kvs) == 0 {
		return nil
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
			logger.Errorf("wrong type of kv mapping value: %T", v)
			return errors.New("wrong type of kv mapping value")
		}
	}
	cancel()
	return nil
}

func (gw *ApiGatewayRegistrant) deleteMany(k interface{}, opts ...clientv3.OpOption) error {
	if gw.cli == nil {
		logger.Error("etcd client need initialize")
		return errors.New("etcd client need initialize")
	}
	cli := gw.cli

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if sliceValue, ok := k.([]string); ok {
		for _, v := range sliceValue {
			_, err := cli.Delete(ctx, v, opts...)
			if err != nil {
				cancel()
				return err
			}
		}
	} else if strValue, ok := k.(string); ok {
		_, err := cli.Delete(ctx, strValue, opts...)
		if err != nil {
			cancel()
			return err
		}
	} else {
		cancel()
		logger.Errorf("wrong type of kv mapping value: %T", k)
		return errors.New("wrong type of kv mapping value")
	}
	cancel()
	return nil
}

func (gw *ApiGatewayRegistrant) getAttr(key string) string {
	resp, err := gw.getKeyValue(key)
	if err != nil {
		logger.Exception(err)
		return ""
	}
	if resp.Count == 0 {
		return ""
	} else if resp.Count == 1 {
		return string(resp.Kvs[0].Value)
	} else {
		logger.Warningf("attr [%s] exists more than one key", key)
		os.Exit(-1)
		return ""
	}
}

func (gw *ApiGatewayRegistrant) deleteInvalidNodeInService() error {

	//delete old node in exist services
	resp, err := gw.getKeyValueWithPrefix(ServiceDefinitionPrefix)
	if err != nil {
		logger.Exception(err)
		return err
	}
	kvs := make(map[string]interface{})
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, ServiceDefinitionBytes)
		tmpSlice := bytes.Split(key, SlashBytes)
		if len(tmpSlice) != 2 {
			logger.Warningf("invalid router definition: %s", key)
			continue
		}
		sName := string(bytes.TrimPrefix(tmpSlice[0], ServicePrefixBytes))
		attr := tmpSlice[1]
		if bytes.Equal(attr, NodeKeyBytes) && gw.service.Name != sName {
			var nodeSlice []string
			err := json.Unmarshal(kv.Value, &nodeSlice)
			if err != nil {
				logger.Exception(err)
				continue
			}
			s := mapset.NewSet()
			for _, node := range nodeSlice {
				s.Add(node)
			}
			if s.Contains(gw.node.ID) {
				s.Remove(gw.node.ID)
				nodeByteSlice, err := json.Marshal(s.ToSlice())
				if err != nil {
					logger.Exception(err)
					continue
				} else {
					kvs[string(kv.Key)] = string(nodeByteSlice)
				}
			}
		}
	}
	err = gw.putMany(kvs)
	if err != nil {
		logger.Exception(err)
		return err
	}

	return err
}

func (gw *ApiGatewayRegistrant) registerNode() error {
	nodeDefinition := fmt.Sprintf(NodeDefinition, gw.node.ID)
	resp, err := gw.getKeyValueWithPrefix(nodeDefinition)
	if err != nil {
		logger.Exception(err)
		return err
	}
	n := gw.node
	kvs := make(map[string]interface{})
	if resp.Count == 0 {
		kvs[nodeDefinition+IDKey] = n.ID
		kvs[nodeDefinition+NameKey] = n.Name
		kvs[nodeDefinition+HostKey] = n.Host
		kvs[nodeDefinition+PortKey] = strconv.FormatInt(int64(n.Port), 10)
		kvs[nodeDefinition+StatusKey] = strconv.FormatUint(uint64(n.Status), 10)
		kvs[nodeDefinition+HealthCheckKey] = n.HC.ID
	} else {
		id := gw.getAttr(nodeDefinition + IDKey)
		name := gw.getAttr(nodeDefinition + NameKey)
		host := gw.getAttr(nodeDefinition + HostKey)
		port := gw.getAttr(nodeDefinition + PortKey)
		healthCheck := gw.getAttr(nodeDefinition + HealthCheckKey)
		if id != n.ID {
			kvs[nodeDefinition+IDKey] = n.ID
		}
		if name != n.Name {
			kvs[nodeDefinition+NameKey] = n.Name
		}
		if host != n.Host {
			kvs[nodeDefinition+HostKey] = n.Host
		}
		setPort := strconv.FormatInt(int64(n.Port), 10)
		if port != setPort {
			kvs[nodeDefinition+PortKey] = setPort
		}
		if healthCheck != n.HC.ID {
			kvs[nodeDefinition+HealthCheckKey] = n.HC.ID
		}
		kvs[nodeDefinition+StatusKey] = "2"
		if len(kvs) > 0 {
			logger.Infof("node keys waiting to be updated: %+v", kvs)
		}
	}

	err = gw.putMany(kvs)
	if err != nil {
		logger.Exception(err)
		return err
	}

	hc := gw.hc
	hcDefinition := fmt.Sprintf(HealthCheckDefinition, hc.ID)
	resp, err = gw.getKeyValueWithPrefix(hcDefinition)
	if err != nil {
		logger.Exception(err)
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
		id := gw.getAttr(hcDefinition + IDKey)
		path := gw.getAttr(hcDefinition + PathKey)
		timeout := gw.getAttr(hcDefinition + TimeoutKey)
		interval := gw.getAttr(hcDefinition + IntervalKey)
		retry := gw.getAttr(hcDefinition + RetryKey)
		retryTime := gw.getAttr(hcDefinition + RetryTimeKey)
		if id != hc.ID {
			kvs[hcDefinition+IDKey] = hc.ID
		}
		if path != hc.Path {
			kvs[hcDefinition+PathKey] = hc.Path
		}
		if timeout != strconv.FormatInt(int64(hc.Timeout), 10) {
			kvs[hcDefinition+TimeoutKey] = hc.Timeout
		}
		if interval != strconv.FormatInt(int64(hc.Interval), 10) {
			kvs[hcDefinition+IntervalKey] = hc.Interval
		}
		tmp := "1"
		if !hc.Retry {
			tmp = "0"
		}
		if retry != tmp {
			kvs[hcDefinition+RetryKey] = tmp
		}
		if retryTime != strconv.FormatInt(int64(hc.RetryTime), 10) {
			kvs[hcDefinition+RetryTimeKey] = hc.RetryTime
		}
		if len(kvs) > 0 {
			logger.Infof("node keys waiting to be updated: %+v", kvs)
		}
	}

	err = gw.putMany(kvs)
	if err != nil {
		logger.Exception(err)
		return err
	}

	return nil
}

func (gw *ApiGatewayRegistrant) registerService() error {
	if err := gw.deleteInvalidNodeInService(); err != nil {
		logger.Exception(err)
		return err
	}

	serviceDefinition := fmt.Sprintf(ServiceDefinition, gw.service.Name)
	resp, err := gw.getKeyValueWithPrefix(serviceDefinition)
	if err != nil {
		logger.Exception(err)
		return err
	}
	kvs := make(map[string]interface{})
	if resp.Count == 0 {
		// service not existed
		var nodes []string
		for _, n := range gw.service.Node {
			nodes = append(nodes, n.ID)
		}
		nodeSlice, err := json.Marshal(nodes)
		if err != nil {
			logger.Exception(err)
			return err
		}

		kvs[serviceDefinition+NameKey] = gw.service.Name
		kvs[serviceDefinition+NodeKey] = string(nodeSlice)
	} else {
		resp, err := gw.getKeyValue(serviceDefinition + NodeKey)
		if err != nil {
			logger.Exception(err)
			return err
		}
		if resp.Count == 0 {
			// a ri e na i, impossible!
			return errors.New(fmt.Sprintf("can not find key: [%s]", serviceDefinition+NodeKey))
		} else if resp.Count > 0 {
			var nodeSlice []string
			err := json.Unmarshal(resp.Kvs[0].Value, &nodeSlice)
			if err != nil {
				logger.Exception(err)
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
					logger.Exception(err)
					os.Exit(-1)
				} else {
					kvs[serviceDefinition+NodeKey] = string(nodeByteSlice)
				}
			}
		}
	}
	err = gw.putMany(kvs)
	if err != nil {
		logger.Exception(err)
		return err
	}
	return nil
}

func (gw *ApiGatewayRegistrant) registerRouter() error {
	if err := gw.deleteInvalidRoutes(); err != nil {
		logger.Exception(err)
		return err
	}

	for _, r := range gw.router {
		var frontend string
		var kvs map[string]interface{}
		var ori map[string]interface{}
		kvs = make(map[string]interface{})
		ori = make(map[string]interface{})
		routerName := fmt.Sprintf(RouterDefinition, r.Name)
		resp, err := gw.getKeyValueWithPrefix(routerName)
		if err != nil {
			logger.Exception(err)
			return err
		}
		if strings.Index(r.Frontend, "/") == 0 {
			frontend = r.Method + "@" + r.Frontend
		} else {
			frontend = r.Method + "@/" + r.Frontend
		}
		if resp.Count != 6 {
			kvs[routerName+IDKey] = r.ID
			kvs[routerName+NameKey] = r.Name
			kvs[routerName+StatusKey] = strconv.FormatUint(uint64(r.Status), 10)
			kvs[routerName+FrontendKey] = frontend
			kvs[routerName+BackendKey] = r.Backend
			kvs[routerName+ServiceKey] = r.Service.Name
		} else {
			for _, kv := range resp.Kvs {
				if bytes.Equal(kv.Key, []byte(routerName+StatusKey)) {
					ori[routerName+StatusKey] = string(kv.Value)
				} else if bytes.Equal(kv.Key, []byte(routerName+IDKey)) {
					if !bytes.Equal(kv.Value, []byte(r.ID)) {
						kvs[routerName+IDKey] = string(kv.Value)
					}
					ori[routerName+IDKey] = r.ID
				} else if bytes.Equal(kv.Key, []byte(routerName+NameKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Name)) {
						kvs[routerName+NameKey] = r.Name
					}
					ori[routerName+NameKey] = kv.Value
				} else if bytes.Equal(kv.Key, []byte(routerName+FrontendKey)) {
					if !bytes.Equal(kv.Value, []byte(frontend)) {
						kvs[routerName+FrontendKey] = frontend
					}
					ori[routerName+FrontendKey] = kv.Value
				} else if bytes.Equal(kv.Key, []byte(routerName+BackendKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Backend)) {
						kvs[routerName+BackendKey] = r.Backend
					}
					ori[routerName+BackendKey] = kv.Value
				} else if bytes.Equal(kv.Key, []byte(routerName+ServiceKey)) {
					if !bytes.Equal(kv.Value, []byte(r.Service.Name)) {
						kvs[routerName+ServiceKey] = r.Service.Name
					}
					ori[routerName+ServiceKey] = kv.Value
				} else {
					logger.Warningf("unrecognized router key: %s", string(kv.Key))
				}
			}
			if _, ok := kvs[routerName+FrontendKey]; ok {
				if ori[routerName+FrontendKey] != kvs[routerName+FrontendKey] {
					// just a new route, but have a duplicated name
					logger.Debugf("original frontend key: %s; current frontend key: %s", string(ori[routerName+FrontendKey].([]byte)), string(kvs[routerName+FrontendKey].([]byte)))
					logger.Warningf("router {%s} maybe a new router, please checkout", string(ori[routerName+NameKey].([]byte)))
					os.Exit(-1)
				}
			}
			if len(kvs) > 0 {
				logger.Infof("router keys waiting to be updated: %+v", kvs)
				if ori[routerName+StatusKey] == "1" {
					// original router still alive, can not modify router
					logger.Warning("original router still alive, can not modify router: ", routerName)
					logger.Warning("if need modify online router, please use a new one instead")
					os.Exit(-1)
				}
			}
		}

		if err = gw.putMany(kvs); err != nil {
			logger.Exception(err)
			return err
		}
	}

	return nil
}

func (gw *ApiGatewayRegistrant) deleteInvalidRoutes() error {
	resp, err := gw.getKeyValueWithPrefix(RouterDefinitionPrefix)
	if err != nil {
		logger.Exception(err)
		return err
	}

	invalidMap := map[string]bool{}
	setRouteMap := map[string]*Router{}
	for _, v := range gw.router {
		setRouteMap[v.Name] = v
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, RouterDefinitionBytes)
		tmpSlice := bytes.Split(key, SlashBytes)
		if len(tmpSlice) != 2 {
			logger.Warningf("invalid router definition: %s", key)
			continue
		}
		rName := string(bytes.TrimPrefix(tmpSlice[0], RouterPrefixBytes))
		attr := tmpSlice[1]
		if bytes.Equal(attr, ServiceKeyBytes) {
			if string(kv.Value) == gw.service.Name {
				if setRouteMap[rName] == nil {
					routerName := fmt.Sprintf(RouterDefinition, rName)
					invalidMap[routerName] = true
				}
			}
		}
	}

	invalidKeys := make([]string, 0, len(invalidMap))
	for k := range invalidMap {
		invalidKeys = append(invalidKeys, k)
		logger.Info("delete invalid router:", k)
	}

	err = gw.deleteMany(invalidKeys, clientv3.WithPrefix())
	if err != nil {
		logger.Exception(err)
		return err
	}
	return nil
}

func (gw *ApiGatewayRegistrant) Register() error {
	if err := gw.registerNode(); err != nil {
		logger.Exception(err)
		return err
	}
	if err := gw.registerService(); err != nil {
		logger.Exception(err)
		return err
	}
	if err := gw.registerRouter(); err != nil {
		logger.Exception(err)
		return err
	}
	return nil
}

func (gw *ApiGatewayRegistrant) Unregister() error {
	var kvs map[string]interface{}
	kvs = make(map[string]interface{})

	nodeDefinition := fmt.Sprintf(NodeDefinition, gw.node.ID)
	resp, err := gw.getKeyValueWithPrefix(nodeDefinition)
	if err != nil {
		logger.Exception(err)
		return err
	}
	if resp.Count == 0 {
		kvs[nodeDefinition+StatusKey] = strconv.FormatUint(uint64(2), 10)
	} else {
		kvs[nodeDefinition+StatusKey] = "2"
	}

	err = gw.putMany(kvs)
	if err != nil {
		logger.Exception(err)
		return err
	}
	return nil
}
