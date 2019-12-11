package watcher

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/hhjpin/goutils/logger"
)

const (
	routeWatcherPrefix       = "/Router/"
	serviceWatcherPrefix     = "/Service/"
	endpointWatcherPrefix    = "/Node/"
	healthCheckWatcherPrefix = "/HealthCheck/"

	slash = "/"
)

func validKV(cli *clientv3.Client, prefix string, attrs []string, not bool) (bool, error) {
	for _, attr := range attrs {
		ctx := context.Background()
		resp, err := cli.Get(ctx, prefix+attr)
		if err != nil {
			logger.Error(err)
			return false, err
		}
		/*for _, kv := range resp.Kvs {
			logger.Debugf("valid kv resp: %s, %s", string(kv.Key), string(kv.Value))
		}*/

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
