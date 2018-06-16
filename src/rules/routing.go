package rules

import (
	"github.com/coreos/etcd/clientv3"
	"sync"
)

type Algorithm struct {

}

type EtcdCache struct {
	cli *clientv3.Client

	mu sync.RWMutex
	table *RoutingTable
}

type RoutingTable struct {
	version uint8

	rule routingRules
}

type routingRules struct {

}

type Api struct {

}
type Service struct {

}

type Router struct {
	name string

	api *Api
	svr *Service

	alg *Algorithm
}
