package watcher

import (
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"github.com/coreos/etcd/clientv3"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/logger"
	"strings"
)

type RouteWatcher struct {
	prefix    string
	attrs     []string
	table     *routing.Table
	WatchChan clientv3.WatchChan
	ctx       context.Context
	cli       *clientv3.Client
}

func NewRouteWatcher(cli *clientv3.Client, ctx context.Context) *RouteWatcher {
	w := &RouteWatcher{
		cli:    cli,
		prefix: routeWatcherPrefix,
		attrs:  []string{"BackendApi", "FrontendApi", "ID", "Name", "Service", "Status"},
		ctx:    ctx,
	}
	w.WatchChan = cli.Watch(ctx, w.prefix, clientv3.WithPrefix())
	return w
}

func (r *RouteWatcher) Ctx() context.Context {
	return r.ctx
}

func (r *RouteWatcher) GetWatchChan() clientv3.WatchChan {
	return r.WatchChan
}

func (r *RouteWatcher) Refresh() {
	r.ctx = context.Background()
	r.WatchChan = r.cli.Watch(r.ctx, r.prefix, clientv3.WithPrefix())
}

func (r *RouteWatcher) Put(key, val string, isCreate bool) error {
	route := strings.TrimPrefix(key, r.prefix+"Router-")
	tmp := strings.Split(route, slash)
	if len(tmp) < 2 {
		logger.Warnf("invalid router key: %s", key)
		return errors.NewFormat(200, fmt.Sprintf("invalid router key: %s", key))
	}
	routeName := tmp[0]
	routeKey := r.prefix + fmt.Sprintf("Router-%s/", routeName)
	logger.Debugf("[ETCD PUT] Router, key: %s, val: %s, new: %t", key, val, isCreate)
	if isCreate {
		if ok, err := validKV(r.cli, routeKey, r.attrs, false); err != nil || !ok {
			logger.Warnf("new route lack attribute, it may not have been created yet. Suggest to wait")
			return nil
		} else {
			if err := r.table.RefreshRouterByName(routeName, routeKey); err != nil {
				logger.Error(err)
				return err
			}
			return nil
		}
	} else {
		if err := r.table.RefreshRouterByName(routeName, routeKey); err != nil {
			logger.Error(err)
			return err
		}
		return nil
	}
}

func (r *RouteWatcher) Delete(key string) error {
	route := strings.TrimPrefix(key, r.prefix+"Router-")
	tmp := strings.Split(route, slash)
	if len(tmp) < 2 {
		logger.Warnf("invalid router key: %s", key)
		return errors.NewFormat(200, fmt.Sprintf("invalid router key: %s", key))
	}
	routeName := tmp[0]
	//routeKey := r.prefix + fmt.Sprintf("Router-%s/", routeName)
	logger.Debugf("新的Router删除事件, name: %s, key: %s", routeName, key)

	//if ok, err := validKV(r.cli, routeKey, r.attrs, true); err != nil || !ok {
	//	logger.Warnf("route attribute still exists, it may not have been deleted yet. Suggest to wait")
	//	return nil
	//} else {
	if err := r.table.DeleteRouter(routeName); err != nil {
		logger.Error(err)
		return err
	}
	return nil
	//}
}

func (r *RouteWatcher) BindTable(table *routing.Table) {
	r.table = table
}

func (r *RouteWatcher) GetTable() *routing.Table {
	return r.table
}
