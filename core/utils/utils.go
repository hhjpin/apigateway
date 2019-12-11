package utils

import (
	"bytes"
	"context"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/middleware"
	"github.com/coreos/etcd/clientv3"
	"github.com/deckarep/golang-set"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/logger"
	"io/ioutil"
	"runtime"
	"time"
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

func GetKV(cli *clientv3.Client, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.NewFormat(200, "etcd client need initialising")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	if err != nil {
		logger.Debugf("GetKV error: %s", err)
	}
	return resp, err
}

func GetPrefixKV(cli *clientv3.Client, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.NewFormat(200, "etcd client need initialising")
	}
	opts = append(opts, clientv3.WithPrefix())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Get(ctx, key, opts...)
	cancel()
	if err != nil {
		logger.Errorf("GetKV error: %s", err.Error())
	}
	return resp, err
}

func PutKV(cli *clientv3.Client, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	if cli == nil {
		logger.Error("etcd client need initialize")
		return nil, errors.NewFormat(200, "etcd client need initialising")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := cli.Put(ctx, key, value, opts...)
	cancel()
	if err != nil {
		logger.Debugf("PutKV error: %s", err)
	}
	return resp, err
}

func PutKVs(cli *clientv3.Client, kv interface{}, opts ...clientv3.OpOption) error {
	if cli == nil {
		logger.Error("etcd client need initialize")
		return errors.NewFormat(200, "etcd client need initialising")
	}

	kvs, ok := kv.(map[string]interface{})
	if !ok {
		return errors.NewFormat(200, "parameter [kv] must be map[string]interface{}")
	}

	for k, v := range kvs {
		logger.Debugf("Put many: {%s: %s}", k, v)
		if sliceValue, ok := v.([]string); ok {
			for _, item := range sliceValue {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := cli.Put(ctx, k, item, opts...)
				cancel()
				if err != nil {
					logger.Debugf("PutKV error: %s", err)
					return err
				}
			}
		} else if strValue, ok := v.(string); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := cli.Put(ctx, k, strValue, opts...)
			cancel()
			if err != nil {
				logger.Debugf("PutKV error: %s", err)
				return err
			}
		} else {
			return errors.NewFormat(200, "only support string or []string")
		}
	}
	return nil
}

func CmpPointerSlice(a, b []*middleware.Middleware) bool {
	var aSet, bSet mapset.Set
	aSet = mapset.NewSet()
	for _, i := range a {
		aSet.Add(i)
	}
	bSet = mapset.NewSet()
	for _, j := range b {
		bSet.Add(j)
	}
	return aSet.Equal(bSet)
}

func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

func source(lines [][]byte, n int) []byte {
	n--
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

func TimeFormat(t time.Time) string {
	var timeString = t.Format("2006/01/02 - 15:04:05")
	return timeString
}

func Stack(skip int) []byte {
	buf := new(bytes.Buffer)

	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		if _, err := fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc); err != nil {
			logger.Error(err)
		}
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		if _, err := fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line)); err != nil {
			logger.Error(err)
		}
	}
	return buf.Bytes()
}
