package core

import (
	"api_gateway/utils"
	"bytes"
	"context"
	"github.com/coreos/etcd/clientv3"
	"log"
	"sync"
	"time"
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
	PathKeyBytes           = []byte("Path")
	TimeoutKeyBytes        = []byte("Timeout")
	IntervalKeyBytes       = []byte("Interval")
	RetryKeyBytes          = []byte("Retry")
	RetryTimeKeyBytes      = []byte("RetryTime")

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

				}
			}
		}
	}
}
