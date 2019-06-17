package watcher

import (
	"context"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
)

const (
	routeWatcherPrefix       = "/Router/"
	serviceWatcherPrefix     = "/Service/"
	endpointWatcherPrefix    = "/Node/"
	healthCheckWatcherPrefix = "/HealthCheck/"

	slash = "/"
)

var (
	logger = log.Logger
)

func validKV(cli *clientv3.Client, prefix string, attrs []string, not bool) (bool, error) {
	for _, attr := range attrs {
		ctx := context.Background()
		resp, err := cli.Get(ctx, prefix+attr)
		if err != nil {
			logger.Exception(err)
			return false, err
		}
		for _, kv := range resp.Kvs {
			logger.Debugf("valid kv resp: %s, %s", string(kv.Key), string(kv.Value))
		}

		if not {
			if len(resp.Kvs) > 0 {
				// key exists
				return false, nil
			}
		} else {
			if len(resp.Kvs) == 0 {
				// key not exists
				return false, nil
			}
		}
	}
	return true, nil
}