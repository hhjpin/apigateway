package watcher

import (
	"bytes"
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"strings"
)

type EndpointWatcher struct {
	prefix    string
	attrs     []string
	table     *routing.Table
	WatchChan clientv3.WatchChan
	ctx context.Context
	cli       *clientv3.Client
}

func NewEndpointWatcher(cli *clientv3.Client, ctx context.Context) *EndpointWatcher {
	ep := &EndpointWatcher{
		cli:    cli,
		prefix: endpointWatcherPrefix,
		attrs:  []string{"ID", "Name", "Port", "Host"},
		ctx: ctx,
	}
	ep.WatchChan = cli.Watch(ctx, ep.prefix, clientv3.WithPrefix())
	return ep
}

func (ep *EndpointWatcher) Ctx() context.Context {
	return ep.ctx
}

func (ep *EndpointWatcher) GetWatchChan() clientv3.WatchChan{
	return ep.WatchChan
}

func (ep *EndpointWatcher) Refresh() {
	ep.ctx = context.Background()
	ep.WatchChan = ep.cli.Watch(ep.ctx, ep.prefix, clientv3.WithPrefix())
}

func (ep *EndpointWatcher) Put(kv *mvccpb.KeyValue, isCreate bool) error {
	endpoint := strings.TrimPrefix(string(kv.Key), ep.prefix+"Node-")
	tmp := strings.Split(endpoint, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid endpoint key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid endpoint key: %s", string(kv.Key)))
	}
	endpointId := tmp[0]
	endpointKey := ep.prefix + fmt.Sprintf("Node-%s/", endpointId)
	if bytes.Equal(kv.Key, []byte(endpointKey + constant.FailedTimesKeyString)) {
		// ignore failed times key put event
		return nil
	}
	logger.Debugf("新的Endpoint写入事件, key: %s, value: %s", string(kv.Key), string(kv.Value))
	if isCreate {
		if ok, err := validKV(ep.cli, endpointKey, ep.attrs, false); err != nil || !ok {
			logger.Warningf("new endpoint lack attribute, it may not have been created yet. Suggest to wait")
			return nil
		} else {
			if err := ep.table.RefreshEndpoint(endpointId, endpointKey); err != nil {
				logger.Exception(err)
				return err
			}
			return nil
		}
	} else {
		if err := ep.table.RefreshEndpoint(endpointId, endpointKey); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}

func (ep *EndpointWatcher) Delete(kv *mvccpb.KeyValue) error {
	endpoint := strings.TrimPrefix(string(kv.Key), ep.prefix+"Node-")
	tmp := strings.Split(endpoint, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid endpoint key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid endpoint key: %s", string(kv.Key)))
	}
	endpointId := tmp[0]
	endpointKey := ep.prefix + fmt.Sprintf("Node-%s/", endpointId)
	logger.Debugf("新的Endpoint删除事件, id: %s, key: %s", endpointId, endpointKey)

	if ok, err := validKV(ep.cli, endpointKey, ep.attrs, true); err != nil || !ok {
		logger.Warningf("endpoint attribute still exists, it may not have been deleted yet. Suggest to wait")
		return nil
	} else {
		if err := ep.table.DeleteEndpoint(endpointId); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}

func (ep *EndpointWatcher) BindTable(table *routing.Table) {
	ep.table = table
}

func (ep *EndpointWatcher) GetTable() *routing.Table {
	return ep.table
}
