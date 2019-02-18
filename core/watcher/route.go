package watcher

import (
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"strings"
)

type RouteWatcher struct {
	prefix    string
	attrs     []string
	table     *routing.Table
	WatchChan clientv3.WatchChan
	cli       *clientv3.Client
}

func NewRouteWatcher(cli *clientv3.Client, ctx context.Context) *RouteWatcher {
	w := &RouteWatcher{
		cli: cli,
		prefix:    routeWatcherPrefix,
		attrs:     []string{"BackendApi", "FrontendApi", "ID", "Name", "Service", "Status"},
	}
	w.WatchChan = cli.Watch(ctx, w.prefix, clientv3.WithPrefix())
	return w
}

func (r *RouteWatcher)Put(kv *mvccpb.KeyValue, isCreate bool) error {
	route := strings.TrimPrefix(string(kv.Key), r.prefix + "Router-")
	tmp := strings.Split(route, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid router key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid router key: %s", string(kv.Key)))
	}
	routeName := tmp[0]
	logger.Debugf("route name: %s", routeName)

	if isCreate {
		if ok, err := validKV(r.cli, r.prefix + fmt.Sprintf("Router-%s/", routeName), r.attrs, false); err != nil || !ok {
			logger.Warningf("new route lack attribute, it may not have been created yet. Suggest to wait")
			return nil
		} else {
			if err := r.RefreshRouter(routeName, mvccpb.PUT); err != nil {
				logger.Exception(err)
				return err
			}
			return nil
		}
	} else {
		if err := r.RefreshRouter(routeName, mvccpb.PUT); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}

func (r *RouteWatcher)Delete(kv *mvccpb.KeyValue) error {
	route := strings.TrimPrefix(string(kv.Key), r.prefix + "Router-")
	tmp := strings.Split(route, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid router key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid router key: %s", string(kv.Key)))
	}
	routeName := tmp[0]
	logger.Debugf("route name: %s", routeName)
	if ok, err := validKV(r.cli, r.prefix + fmt.Sprintf("Router-%s/", routeName), r.attrs, true); err != nil || !ok {
		logger.Warningf("route attribute still exists, it may not have been deleted yet. Suggest to wait")
		return nil
	} else {
		if err := r.RefreshRouter(routeName, mvccpb.DELETE); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}

func (r *RouteWatcher)BindTable(table *routing.Table) {
	r.table = table
}

func (r *RouteWatcher)GetTable() *routing.Table {
	return r.table
}

func (r *RouteWatcher)RefreshRouter(name string, eventType mvccpb.Event_EventType) error {
	switch eventType {
	case mvccpb.PUT:
		if err := r.table.RefreshRouter(name); err != nil {
			logger.Exception(err)
			return err
		}
	case mvccpb.DELETE:
		if err := r.table.DeleteRouter(name); err != nil {
			logger.Exception(err)
			return err
		}
	default:
		logger.Errorf("unrecognized event type: %d", eventType)
		return errors.NewFormat(200, fmt.Sprintf("unrecognized event type: %d", eventType))
	}
	return nil
}

