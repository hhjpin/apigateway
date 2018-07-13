package core

import (
	"api_gateway/utils"
	"github.com/coreos/etcd/clientv3"
	"log"
	"sync"
	"time"
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
