package core

import (
	"context"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/coreos/etcd/clientv3"
	"time"
)

func (r *RoutingTable) getKeyValue(key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if r.cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.NewFormat(200, "etcd client need initialising")
	}
	cli := r.cli
	logger.Debugf("Get key: %s", key)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	return resp, err
}

func (r *RoutingTable) getKeyValueWithPrefix(key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if r.cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.NewFormat(200, "etcd client need initialising")
	}
	cli := r.cli
	logger.Debugf("Get key: %s with prefix", key)
	opts = append(opts, clientv3.WithPrefix())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	return resp, err
}

func (r *RoutingTable) putKeyValue(key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	if r.cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.NewFormat(200, "etcd client need initialising")
	}
	cli := r.cli
	logger.Debugf("Put kv: {%s: %s}", key, value)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Put(ctx, key, value, opts...)
	cancel()
	return resp, err
}

func (r *RoutingTable) putMany(kv interface{}, opts ...clientv3.OpOption) error {
	if r.cli == nil {
		logger.Error("etcd client need initialize")
		return errors.NewFormat(200, "etcd client need initialising")
	}
	cli := r.cli

	kvs, ok := kv.(map[string]interface{})
	if !ok {
		return errors.NewFormat(200, "parameter [kv] must be map[string]interface{}")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	for k, v := range kvs {
		logger.Debugf("Put many: {%s: %s}", k, v)
		if sliceValue, ok := v.([]string); ok {
			for _, item := range sliceValue {
				_, err := cli.Put(ctx, k, item, opts...)
				if err != nil {
					cancel()
					return err
				}
			}
		} else if strValue, ok := v.(string); ok {
			_, err := cli.Put(ctx, k, strValue, opts...)
			if err != nil {
				cancel()
				return err
			}
		} else {
			cancel()
			return errors.NewFormat(200, "interface{} only support string or []string")
		}
	}
	cancel()
	return nil
}
