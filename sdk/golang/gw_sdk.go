package golang

import (
	"github.com/coreos/etcd/clientv3"
	"log"
	"context"
	"time"
	"fmt"
	"errors"
	"encoding/json"
)

type Node struct{
	ID string
	Name string
	Host string
	Port uint8
	Status uint8
	HC *HealthCheck
}
type Service struct{
	Name string
	Node []*Node
}
type Router struct{
	ID string
	Name string
	Frontend string
	Backend string
	Service *Service
}
type HealthCheck struct{
	ID string
	Path string
	Timeout uint8
	Interval uint8
	Retry bool
	RetryTime uint8
}

type ApiGatewayRegistrant struct {
	cli *clientv3.Client

	node *Node
	service *Service
	hc *HealthCheck
	router []*Router
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
		Retry:    	retry,
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
		Name:     name + uid,
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
	var kvs map[string][]string
	kvs = make(map[string][]string)

	_, err := gw.getKeyValueWithPrefix(fmt.Sprintf(NodeDefinition, gw.node.ID))
	if err != nil {
		log.Print(err)
		return err
	}
	kvs[fmt.Sprintf(NodeDefinition, gw.node.ID) + IDKey] = []string{
		gw.node.ID,
		gw.node.Name,
		gw.node.Host,
		string(gw.node.Port),
		string(gw.node.Status),
		gw.node.HC.ID,
	}

	err = gw.putMany(kvs)
	if err != nil {
		log.Print(err)
	}

	return nil
}

func (gw *ApiGatewayRegistrant) registerService() error {
	var kvs map[string][]string
	kvs = make(map[string][]string)

	resp, err := gw.getKeyValueWithPrefix(fmt.Sprintf(ServiceDefinition, gw.service.Name))
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

		kvs[fmt.Sprintf(ServiceDefinition, gw.service.Name)] = []string{
			gw.service.Name,
			string(nodeSlice),
		}
	} else {
		resp, err := gw.getKeyValue(fmt.Sprintf(ServiceDefinition, gw.service.Name) + NodeKey)
		if err != nil {
			log.Print(err)
			return err
		}
		if resp.Count == 0 {
			// a ri e na i, impossible!
			return errors.New(fmt.Sprintf("can not find key: [%s]", fmt.Sprintf(ServiceDefinition, gw.service.Name) + NodeKey))
		} else {

		}
	}
	return nil
}

