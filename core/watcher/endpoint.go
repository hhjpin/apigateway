package watcher

import (
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"github.com/coreos/etcd/clientv3"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/logger"
	"strings"
)

type EndpointWatcher struct {
	prefix    string
	attrs     []string
	table     *routing.Table
	WatchChan clientv3.WatchChan
	ctx       context.Context
	cli       *clientv3.Client
}

func NewEndpointWatcher(cli *clientv3.Client, ctx context.Context) *EndpointWatcher {
	ep := &EndpointWatcher{
		cli:    cli,
		prefix: endpointWatcherPrefix,
		attrs:  []string{"ID", "Name", "Port", "Host"},
		ctx:    ctx,
	}
	ep.WatchChan = cli.Watch(ctx, ep.prefix, clientv3.WithPrefix())
	return ep
}

func (ep *EndpointWatcher) Ctx() context.Context {
	return ep.ctx
}

func (ep *EndpointWatcher) GetWatchChan() clientv3.WatchChan {
	return ep.WatchChan
}

func (ep *EndpointWatcher) Refresh() {
	ep.ctx = context.Background()
	ep.WatchChan = ep.cli.Watch(ep.ctx, ep.prefix, clientv3.WithPrefix())
}

func (ep *EndpointWatcher) Put(key, val string, isCreate bool) error {
	endpoint := strings.TrimPrefix(key, ep.prefix+"Node-")
	tmp := strings.Split(endpoint, slash)
	if len(tmp) < 2 {
		logger.Warnf("invalid endpoint key: %s", key)
		return errors.NewFormat(200, fmt.Sprintf("invalid endpoint key: %s", key))
	}
	endpointId := tmp[0]
	endpointKey := ep.prefix + fmt.Sprintf("Node-%s/", endpointId)
	if key == endpointKey+constant.FailedTimesKeyString {
		// ignore failed times key put event
		return nil
	}
	logger.Debugf("[ETCD PUT] Endpoint, key: %s, value: %s, new: %t", key, val, isCreate)
	if isCreate {
		if ok, err := validKV(ep.cli, endpointKey, ep.attrs, false); err != nil || !ok {
			logger.Warnf("new endpoint lack attribute, it may not have been created yet. Suggest to wait")
			return nil
		} else {
			if err := ep.table.RefreshEndpointById(endpointId, endpointKey); err != nil {
				logger.Error(err)
				return err
			}
			return nil
		}
	} else {
		if err := ep.table.RefreshEndpointById(endpointId, endpointKey); err != nil {
			logger.Error(err)
			return err
		}
		return nil
	}
}

func (ep *EndpointWatcher) Delete(key string) error {
	endpoint := strings.TrimPrefix(key, ep.prefix+"Node-")
	tmp := strings.Split(endpoint, slash)
	if len(tmp) < 2 {
		logger.Warnf("invalid endpoint key: %s", key)
		return errors.NewFormat(200, fmt.Sprintf("invalid endpoint key: %s", key))
	}
	endpointId := tmp[0]
	//endpointKey := ep.prefix + fmt.Sprintf("Node-%s/", endpointId)
	logger.Debugf("[ETCD DELETE] Endpoint key: %s", key)

	/*if ok, err := validKV(ep.cli, endpointKey, ep.attrs, true); err != nil || !ok {
		logger.Warnf("endpoint attribute still exists, it may not have been deleted yet. Suggest to wait")
		return nil
	} else {*/
	if err := ep.table.DeleteEndpoint(endpointId); err != nil {
		logger.Error(err)
		return err
	}
	return nil
	//}
}

func (ep *EndpointWatcher) BindTable(table *routing.Table) {
	ep.table = table
}

func (ep *EndpointWatcher) GetTable() *routing.Table {
	return ep.table
}
