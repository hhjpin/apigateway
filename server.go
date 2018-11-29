package main

import (
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
	"github.com/valyala/fasthttp"
	"os"
	"sync"
	"time"
)

type etcdPool struct {
	sync.RWMutex
	internal map[string]*clientv3.Client
}

var (
	table    *core.RoutingTable
	EtcdPool = etcdPool{}

	eLogger = log.New()
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
	key := Conf.Etcd.Name
	conf := Conf.Etcd

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
			eLogger.Exception(err)
			os.Exit(-1)
		}
		return cli
	}
}

func init() {
	etcdCli := ConnectToEtcd()
	table = core.InitRoutingTable(etcdCli)

	//ctx := context.Background()
	//ch := etcdCli.Watch(ctx, "/Router", clientv3.WithPrefix())

	//go core.RouterWatcher(ch)
}

func main() {
	var server *fasthttp.Server
	server = &fasthttp.Server{
		Handler: core.MainRequestHandlerWrapper(table, middleware.Limiter),

		Name:               Conf.Server.Name,
		Concurrency:        Conf.Server.Concurrency,
		ReadBufferSize:     Conf.Server.ReadBufferSize,
		WriteBufferSize:    Conf.Server.WriteBufferSize,
		DisableKeepalive:   Conf.Server.DisabledKeepAlive,
		ReduceMemoryUsage:  Conf.Server.ReduceMemoryUsage,
		MaxRequestBodySize: Conf.Server.MaxRequestBodySize,
	}

	host := fmt.Sprintf("%s:%d", Conf.Server.ListenHost, Conf.Server.ListenPort)
	eLogger.Info(host)
	err := server.ListenAndServe(host)
	if err != nil {
		eLogger.Exception(err)
		os.Exit(-1)
	}
}
