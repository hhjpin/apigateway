package watcher

import (
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"strings"
)

type HealthCheckWatcher struct {
	prefix    string
	attrs     []string
	table     *routing.Table
	WatchChan clientv3.WatchChan
	ctx context.Context
	cli       *clientv3.Client
}

func NewHealthCheckWatcher(cli *clientv3.Client, ctx context.Context) *HealthCheckWatcher {
	hc := &HealthCheckWatcher{
		cli:    cli,
		prefix: healthCheckWatcherPrefix,
		attrs:  []string{"ID", "Interval", "Path", "Retry", "RetryTime", "RetryTime"},
		ctx: ctx,
	}
	hc.WatchChan = cli.Watch(ctx, hc.prefix, clientv3.WithPrefix())
	return hc
}


func (hc *HealthCheckWatcher) Ctx() context.Context {
	return hc.ctx
}

func (hc *HealthCheckWatcher) GetWatchChan() clientv3.WatchChan{
	return hc.WatchChan
}

func (hc *HealthCheckWatcher) Refresh() {
	hc.ctx = context.Background()
	hc.WatchChan = hc.cli.Watch(hc.ctx, hc.prefix, clientv3.WithPrefix())
}

func (hc *HealthCheckWatcher) Put(kv *mvccpb.KeyValue, isCreate bool) error {
	healthCheck := strings.TrimPrefix(string(kv.Key), hc.prefix+"HC-")
	tmp := strings.Split(healthCheck, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid healthCheck key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid healthCheck key: %s", string(kv.Key)))
	}
	hcId := tmp[0]
	hcKey := hc.prefix + fmt.Sprintf(constant.HealthCheckPrefixString, hcId)
	logger.Infof("新的HealthCheck写入事件")
	logger.Debugf("healthCheck id: %s", hcId)
	logger.Debugf("healthCheck key: %s", hcKey)

	if isCreate {
		if ok, err := validKV(hc.cli, hcKey, hc.attrs, false); err != nil || !ok {
			logger.Warningf("new healthCheck lack attribute, it may not have been created yet. Suggest to wait")
			return nil
		} else {
			if err := hc.table.RefreshHealthCheck(hcId, hcKey); err != nil {
				logger.Exception(err)
				return err
			}
			return nil
		}
	} else {
		if err := hc.table.RefreshHealthCheck(hcId, hcKey); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}

func (hc *HealthCheckWatcher) Delete(kv *mvccpb.KeyValue) error {
	healthCheck := strings.TrimPrefix(string(kv.Key), hc.prefix+"HC-")
	tmp := strings.Split(healthCheck, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid healthCheck key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid healthCheck key: %s", string(kv.Key)))
	}
	hcId := tmp[0]
	hcKey := hc.prefix + fmt.Sprintf(constant.HealthCheckPrefixString, hcId)
	logger.Infof("新的HealthCheck删除事件")
	logger.Debugf("healthCheck id: %s", hcId)
	logger.Debugf("healthCheck key: %s", hcKey)

	logger.Infof("HealthCheck delete event will not delete healthCheck object")
	return nil
}

func (hc *HealthCheckWatcher) BindTable(table *routing.Table) {
	hc.table = table
}

func (hc *HealthCheckWatcher) GetTable() *routing.Table {
	return hc.table
}
