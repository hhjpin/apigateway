package main

import (
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/conf"
	"git.henghajiang.com/backend/api_gateway_v2/core"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
	"github.com/valyala/fasthttp"
	"os"
	"runtime"
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

	logger = log.New()
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
	key := conf.Conf.Etcd.Name
	config := conf.Conf.Etcd

	cli, exists := EtcdPool.Load(key)
	if exists {
		return cli
	} else {
		cli, err := clientv3.New(
			clientv3.Config{
				Endpoints:            config.Endpoints,
				AutoSyncInterval:     time.Duration(config.AutoSyncInterval) * time.Second,
				DialTimeout:          time.Duration(config.DialTimeout) * time.Second,
				DialKeepAliveTime:    time.Duration(config.DialKeepAliveTime) * time.Second,
				DialKeepAliveTimeout: time.Duration(config.DialKeepAliveTimeout) * time.Second,
				Username:             config.Username,
				Password:             config.Password,
			},
		)
		if err != nil {
			logger.Exception(err)
			os.Exit(-1)
		}
		return cli
	}
}

func init() {
	etcdCli := ConnectToEtcd()
	table = core.InitRoutingTable(etcdCli)
	ctx := context.Background()
	ch := etcdCli.Watch(ctx, "/Router", clientv3.WithPrefix())

	go table.HealthCheck()
	go core.RouterWatcher(ch)
}

func main() {
	var server *fasthttp.Server

	runtime.GOMAXPROCS(runtime.NumCPU())

	serverConf := conf.Conf.Server
	server = &fasthttp.Server{
		Handler: core.MainRequestHandlerWrapper(table, middleware.Limiter),

		Name:               serverConf.Name,
		Concurrency:        serverConf.Concurrency,
		ReadBufferSize:     serverConf.ReadBufferSize,
		WriteBufferSize:    serverConf.WriteBufferSize,
		DisableKeepalive:   serverConf.DisabledKeepAlive,
		ReduceMemoryUsage:  serverConf.ReduceMemoryUsage,
		MaxRequestBodySize: serverConf.MaxRequestBodySize,
	}

	host := fmt.Sprintf("%s:%d", serverConf.ListenHost, serverConf.ListenPort)
	logger.Infof("gateway server start at: %s", host)
	err := server.ListenAndServe(host)
	if err != nil {
		logger.Exception(err)
		os.Exit(-1)
	}
}
