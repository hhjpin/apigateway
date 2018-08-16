package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/deckarep/golang-set"
	"log"
	"time"
	"bytes"
)

type Node struct {
	ID     string
	Name   string
	Host   string
	Port   uint8
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
	Status 	 uint8
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

func NewHealthCheck(path string, timeout, interval, retryTime uint8, retry bool) *HealthCheck {
	var uid string

	hardwareAddr := GetHardwareAddressAsLong()
	if len(hardwareAddr) > 0 {
		// use hardware address of first interface
		uid = string(hardwareAddr[0])
	} else {
		log.Fatal("can not gain local hardware address")
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

func NewNode(name, host string, port uint8, hc *HealthCheck) *Node {
	var uid string

	hardwareAddr := GetHardwareAddressAsLong()
	if len(hardwareAddr) > 0 {
		// use hardware address of first interface
		uid = string(hardwareAddr[0])
	} else {
		log.Fatal("can not gain local hardware address")
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
		uid = string(hardwareAddr[0])
	} else {
		log.Fatal("can not gain local hardware address")
	}

	return &Router{
		ID:       uid,
		Name:     name,
		Status: 0,
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
		log.Fatal("etcd client need initialize")
	}
	cli := gw.cli

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	return resp, err
}

func (gw *ApiGatewayRegistrant) getKeyValueWithPrefix(key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if gw.cli == nil {
		log.Fatal("etcd client need initialize")
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
		log.Fatal("etcd client need initialize")
	}
	cli := gw.cli

	opts = append(opts, clientv3.WithPrefix())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Put(ctx, key, value, opts...)
	cancel()
	return resp, err
}

func (gw *ApiGatewayRegistrant) putMany(kvs interface{}, opts ...clientv3.OpOption) error {
	if gw.cli == nil {
		log.Fatal("etcd client need initialize")
	}
	cli := gw.cli

	opts = append(opts, clientv3.WithPrefix())
	kvs, ok := kvs.(map[string]interface{})
	if !ok {
		return errors.New("wrong type of kv mapping")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	for k, v := range kvs.(map[string]interface{}) {
		if value, ok := v.([]string); ok {
			for _, item := range value {
				_, err := cli.Put(ctx, k, item, opts...)
				if err != nil {
					cancel()
					return err
				}
			}
		} else if value, ok := v.(string); ok {
			_, err := cli.Put(ctx, k, value, opts...)
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
	var kvs map[string]string
	kvs = make(map[string]string)

	nodeDefinition := fmt.Sprintf(NodeDefinition, gw.node.ID)
	_, err := gw.getKeyValueWithPrefix(nodeDefinition)
	if err != nil {
		log.Print(err)
		return err
	}
	kvs[nodeDefinition+IDKey] = gw.node.ID
	kvs[nodeDefinition+NameKey] = gw.node.Name
	kvs[nodeDefinition+HostKey] = gw.node.Host
	kvs[nodeDefinition+PortKey] = string(gw.node.Port)
	kvs[nodeDefinition+StatusKey] = string(gw.node.Status)
	kvs[nodeDefinition+HealthCheckKey] = gw.node.HC.ID

	err = gw.putMany(kvs)
	if err != nil {
		log.Print(err)
		return err
	}

	hcDefinition := fmt.Sprintf(HealthCheckDefinition, gw.hc.ID)
	_, err = gw.getKeyValueWithPrefix(hcDefinition)
	if err != nil {
		log.Print(err)
		return err
	}
	kvs = make(map[string]string)
	kvs[hcDefinition+IDKey] = gw.hc.ID
	kvs[hcDefinition+PathKey] = gw.hc.Path
	kvs[hcDefinition+TimeoutKey] = string(gw.hc.Timeout)
	kvs[hcDefinition+IntervalKey] = string(gw.hc.Interval)
	kvs[hcDefinition+TimeoutKey] = string(gw.hc.Timeout)
	if gw.hc.Retry {
		kvs[hcDefinition+RetryKey] = "1"
	} else {
		kvs[hcDefinition+RetryKey] = "0"
	}
	kvs[hcDefinition+RetryTimeKey] = string(gw.hc.RetryTime)
	err = gw.putMany(kvs)
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}

func (gw *ApiGatewayRegistrant) registerService() error {
	var kvs map[string]string
	kvs = make(map[string]string)

	serviceDefinition := fmt.Sprintf(ServiceDefinition, gw.service.Name)
	resp, err := gw.getKeyValueWithPrefix(serviceDefinition)
	if err != nil {
		log.Print(err)
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
			log.Print(err)
			return err
		}

		kvs[serviceDefinition+NameKey] = gw.service.Name
		kvs[serviceDefinition+NodeKey] = string(nodeSlice)
	} else {
		resp, err := gw.getKeyValue(serviceDefinition + NodeKey)
		if err != nil {
			log.Print(err)
			return err
		}
		if resp.Count == 0 {
			// a ri e na i, impossible!
			return errors.New(fmt.Sprintf("can not find key: [%s]", serviceDefinition+NodeKey))
		} else if resp.Count == 1 {
			var nodeSlice []string
			err := json.Unmarshal(resp.Kvs[0].Value, &nodeSlice)
			if err != nil {
				log.Print(err)
				return err
			}
			s := mapset.NewSet()
			for _, node := range nodeSlice {
				s.Add(node)
			}
			s.Add(gw.node.ID)
			nodeByteSlice, err := json.Marshal(s.ToSlice())

			kvs[serviceDefinition+NameKey] = gw.service.Name
			kvs[serviceDefinition+NodeKey] = string(nodeByteSlice)
		}
	}
	err = gw.putMany(kvs)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func (gw *ApiGatewayRegistrant) registerRouter() error {
	for _, r := range gw.router {
		var kvs map[string]string
		kvs = make(map[string]string)

		routerName := fmt.Sprintf(RouterDefinition, r.Name)
		resp, err := gw.getKeyValueWithPrefix(routerName)
		if err != nil {
			log.Print(err)
			return err
		}
		if resp.Count == 0 {
			kvs[routerName + IDKey] = r.ID
			kvs[routerName + NameKey] = r.Name
			kvs[routerName + StatusKey] = string(r.Status)
			kvs[routerName + FrontendKey] = r.Frontend
			kvs[routerName + BackendKey] = r.Backend
			kvs[routerName + ServiceKey] = r.Service.Name
		} else {
			var flag uint8
			for _, kv := range resp.Kvs {
				if bytes.Equal(kv.Key, []byte(routerName + StatusKey)) {
					if string(kv.Value) != "1" {
						// router still online, can not be updated
						flag = 1
						break
					}
				}
			}
			if flag == 0 {
				kvs[routerName + IDKey] = r.ID
				kvs[routerName + NameKey] = r.Name
				kvs[routerName + StatusKey] = string(r.Status)
				kvs[routerName + FrontendKey] = r.Frontend
				kvs[routerName + BackendKey] = r.Backend
				kvs[routerName + ServiceKey] = r.Service.Name
			}
		}
		err = gw.putMany(kvs)
		if err != nil {
			log.Print(err)
			return err
		}
	}

	return nil
}
