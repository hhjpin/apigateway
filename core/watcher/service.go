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

type ServiceWatcher struct {
	prefix    string
	attrs     []string
	table     *routing.Table
	WatchChan clientv3.WatchChan
	cli       *clientv3.Client
}

func NewServiceWatcher(cli *clientv3.Client, ctx context.Context) *ServiceWatcher {
	s := &ServiceWatcher{
		cli:    cli,
		prefix: serviceWatcherPrefix,
		attrs:  []string{"Name", "Node"},
	}
	s.WatchChan = cli.Watch(ctx, s.prefix, clientv3.WithPrefix())
	return s
}

func (s *ServiceWatcher) BindTable(table *routing.Table) {
	s.table = table
}

func (s *ServiceWatcher) GetTable() *routing.Table {
	return s.table
}

func (s *ServiceWatcher) Put(kv *mvccpb.KeyValue, isCreate bool) error {
	svr := strings.TrimPrefix(string(kv.Key), s.prefix+"Service-")
	tmp := strings.Split(svr, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid service key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid service key: %s", string(kv.Key)))
	}
	svrName := tmp[0]
	svrKey := s.prefix + fmt.Sprintf(constant.ServicePrefixString, svrName)
	logger.Debugf("service name: %s", svrName)
	logger.Debugf("service key: %s", svrKey)

	if isCreate {
		if ok, err := validKV(s.cli, svrKey, s.attrs, false); err != nil || !ok {
			logger.Warningf("new service lack attribute, it may not have been created yet. Suggest to wait")
			return nil
		} else {
			if err := s.table.RefreshService(svrName, svrKey); err != nil {
				logger.Exception(err)
				return err
			}
			return nil
		}
	} else {
		if err := s.table.RefreshService(svrName, svrKey); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}

func (s *ServiceWatcher) Delete(kv *mvccpb.KeyValue) error {
	svr := strings.TrimPrefix(string(kv.Key), s.prefix+"Service-")
	tmp := strings.Split(svr, slash)
	if len(tmp) < 2 {
		logger.Warningf("invalid service key: %s", string(kv.Key))
		return errors.NewFormat(200, fmt.Sprintf("invalid service key: %s", string(kv.Key)))
	}
	svrName := tmp[0]
	svrKey := s.prefix + fmt.Sprintf(constant.ServicePrefixString, svrName)
	logger.Debugf("service name: %s", svrName)
	logger.Debugf("service key: %s", svrKey)
	if ok, err := validKV(s.cli, svrKey, s.attrs, true); err != nil || !ok {
		logger.Warningf("service attribute still exists, it may not have been deleted yet. Suggest to wait")
		return nil
	} else {
		if err := s.table.DeleteService(svrName); err != nil {
			logger.Exception(err)
			return err
		}
		return nil
	}
}
