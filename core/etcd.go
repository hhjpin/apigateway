package core

import (
	"api_gateway/utils"
	"bytes"
	"context"
	"github.com/coreos/etcd/clientv3"
	"log"
	"sync"
	"time"
	"strconv"
	"api_gateway/utils/errors"
)

const (
	Root              = "/"
	Slash             = "/"
	NodeKey           = "Node"
	NodePrefix        = "Node-"
	ServiceKey        = "Service"
	ServicePrefix     = "Service-"
	RouterKey         = "Router"
	RouterPrefix      = "Router-"
	HealthCheckKey    = "HealthCheck"
	HealthCheckPrefix = "HC-"
	IdKey             = "ID"
	NameKey           = "Name"
	PathKey           = "Path"
	TimeoutKey        = "Timeout"
	IntervalKey       = "Interval"
	RetryKey          = "Retry"
	RetryTimeKey      = "RetryTime"

	ServiceDefinition = "/Service/"
	NodePrefixDefinition = "/Node/Node-"
	HealthCheckPrefixDefinition = "/HealthCheck/HC-"
)

var (
	RootBytes              = []byte("/")
	SlashBytes             = []byte("/")
	NodeKeyBytes           = []byte("Node")
	NodePrefixBytes        = []byte("Node-")
	ServiceKeyBytes        = []byte("Service")
	ServicePrefixBytes     = []byte("Service-")
	RouterKeyBytes         = []byte("Router")
	RouterPrefixBytes      = []byte("Router-")
	HealthCheckKeyBytes    = []byte("HealthCheck")
	HealthCheckPrefixBytes = []byte("HC-")
	IdKeyBytes             = []byte("ID")
	NameKeyBytes           = []byte("Name")
	HostKeyBytes = []byte("Host")
	PortKeyBytes = []byte("Port")
	StatusKeyBytes = []byte("Status")
	PathKeyBytes           = []byte("Path")
	TimeoutKeyBytes        = []byte("Timeout")
	IntervalKeyBytes       = []byte("Interval")
	RetryKeyBytes          = []byte("Retry")
	RetryTimeKeyBytes      = []byte("RetryTime")

	NodeDefinitionBytes = []byte("/Node/")
	ServiceDefinitionBytes = []byte("/Service/")

)

type etcdPool struct {
	sync.RWMutex
	internal map[string]*clientv3.Client
}

var (
	EtcdPool = etcdPool{}
)

func (p *etcdPool) Load(key string) (cli *clientv3.Client, exists bool) {
	p.RLock()
	cli, exists = p.internal[key]
	p.RUnlock()
	return cli, exists
}

func (p *etcdPool) Store(key string, value *clientv3.Client) {
	p.Lock()
	p.internal[key] = value
	p.Unlock()
}

func (p *etcdPool) Delete(key string) {
	p.Lock()
	delete(p.internal, key)
	p.Unlock()
}

func ConnectToEtcd() *clientv3.Client {
	key := utils.Conf.Etcd.Name
	conf := utils.Conf.Etcd

	cli, exists := EtcdPool.Load(key)
	if exists {
		return cli
	} else {
		cli, err := clientv3.New(
			clientv3.Config{
				Endpoints:            conf.Endpoints,
				AutoSyncInterval:     time.Duration(conf.AutoSyncInterval) * time.Second,
				DialTimeout:          time.Duration(conf.DialTimeout) * time.Second,
				DialKeepAliveTime:    time.Duration(conf.DialKeepAliveTime) * time.Second,
				DialKeepAliveTimeout: time.Duration(conf.DialKeepAliveTimeout) * time.Second,
				Username:             conf.Username,
				Password:             conf.Password,
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		return cli
	}
}

func InitRoutingTable() *RoutingTable {

	return nil
}

func initServiceNode(cli *clientv3.Client) (ServiceTableMap, EndpointTableMap) {
	var svr ServiceTableMap
	var ep EndpointTableMap

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, ServiceDefinition, clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, ServiceDefinitionBytes)
		tmp := bytes.Split(key, SlashBytes)
		if len(tmp) != 2 {
			log.Printf("invalid service definition: %s", key)
			continue
		} else {
			sName := bytes.TrimPrefix(tmp[0], ServicePrefixBytes)
			if s, ok := svr.Load(ServiceNameString(sName)); ok {
				// service exists
				if bytes.Equal(tmp[1], NameKeyBytes) {
					s.name = kv.Value
					s.nameString = ServiceNameString(kv.Value)
				} else if bytes.Equal(tmp[1], NodeKeyBytes){
					// check node
				} else {
					log.Printf("invalid service attribute: %s", tmp[1])
				}
			} else {
				if bytes.Equal(tmp[1], NameKeyBytes) {
					if bytes.Equal(kv.Value, sName) {
						s = &Service{
							name: sName,
							nameString: ServiceNameString(sName),
							ep: nil,
							onlineEp: nil,
						}
						svr.Store(ServiceNameString(sName), s)
					} else {
						log.Printf("invalid service name, key: %s", string(kv.Key))
					}
				} else if bytes.Equal(tmp[1], NodeKeyBytes) {
					s = &Service{
						name: sName,
						nameString: ServiceNameString(sName),
						ep: nil,
						onlineEp: nil,
					}
					
				} else {

				}
			}
		}
	}
}

func initEndpointNode(cli *clientv3.Client, nodeID string) (*Endpoint, error) {
	var ep Endpoint

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, NodePrefixDefinition + nodeID)
	cancel()
	if err != nil {
		log.Print(err)
		return nil, err
	}
	for _, kv := range resp.Kvs {
		key := bytes.TrimPrefix(kv.Key, []byte(NodePrefixDefinition + nodeID + Slash))
		if bytes.Equal(key, IdKeyBytes) {
			// do nothing
		} else if bytes.Equal(key, NameKeyBytes) {
			ep.name = kv.Value
			ep.nameString = EndpointNameString(kv.Value)
		} else if bytes.Equal(key, HostKeyBytes) {
			ep.host = kv.Value
		} else if bytes.Equal(key, PortKeyBytes) {
			tmpInt, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				log.Print(err)
				return nil, err
			}
			ep.port = uint8(tmpInt)
		} else if bytes.Equal(key, StatusKeyBytes) {
			tmpInt, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				log.Print(err)
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
				return nil, errors.New(50)
			}
		} else if bytes.Equal(key, HealthCheckKeyBytes) {
			// get health check info
			ctxA, cancelA := context.WithTimeout(context.Background(), 1*time.Second)
			respA, err := cli.Get(ctxA, HealthCheckPrefixDefinition + string(kv.Value))
			cancelA()
			if err != nil {
				log.Print(err)
				return nil, err
			}

			var hc HealthCheck
			for _, kvA := range respA.Kvs {
				keyA := bytes.TrimPrefix(kvA.Key, []byte(HealthCheckPrefixDefinition + string(kv.Value) + Slash))
				if bytes.Equal(keyA, IdKeyBytes) {
					// do nothing
				} else if bytes.Equal(keyA, PathKeyBytes) {
					hc.path = kvA.Value
				} else if bytes.Equal(keyA, TimeoutKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10 ,64)
					if err != nil {
						log.Print(err)
						return nil, err
					}
					hc.timeout = uint8(tmpInt)
				} else if bytes.Equal(keyA, IntervalKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10 ,64)
					if err != nil {
						log.Print(err)
						return nil, err
					}
					hc.interval = uint8(tmpInt)
				} else if bytes.Equal(keyA, RetryKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10 ,64)
					if err != nil {
						log.Print(err)
						return nil, err
					}
					if tmpInt == 0 {
						hc.retry = false
					} else {
						hc.retry = true
					}
				} else if bytes.Equal(keyA, RetryTimeKeyBytes) {
					tmpInt, err := strconv.ParseUint(string(kvA.Value), 10 ,64)
					if err != nil {
						log.Print(err)
						return nil, err
					}
					hc.retryTime = uint8(tmpInt)
				} else {
					// unrecognized attribute
					log.SetPrefix("[WARNING]")
					log.Printf("unrecognized health check attribute\n\tkey: %s\n\tvalue: %s", string(kvA.Key), string(kvA.Value))
				}
			}
			ep.healthCheck = &hc
		} else {
			log.SetPrefix("[WARNING]")
			log.Printf("unrecognized node attribute\n\tkey: %s\n\tvalue: %s", string(kv.Key), string(kv.Value))
		}
	}

	if ep.nameString == "" || ep.host == nil || ep.port == 0 {
		log.SetPrefix("[ERROR]")
		log.Printf("endpoint initialized failed. uncompleted attribute assigned. %+v", ep)
		return nil, errors.New(51)
	}
	return &ep, nil
}

