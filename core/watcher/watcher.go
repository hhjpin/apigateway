package watcher

import (
	"context"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/hhjpin/goutils/logger"
	"time"
)

type Watcher interface {
	Put(key, val string, isCreate bool) error
	Delete(key string) error
	BindTable(table *routing.Table)
	GetTable() *routing.Table
	GetWatchChan() clientv3.WatchChan
	Ctx() context.Context
	Refresh()
}

var Mapping map[Watcher]clientv3.WatchChan

func watch(w Watcher, c clientv3.WatchChan) {
	defer func() {
		if err := recover(); err != nil {
			stack := utils.Stack(3)
			logger.Errorf("[Recovery] %s panic recovered:\n%s\n%s", utils.TimeFormat(time.Now()), err, stack)
		}
		// restart watch func
		go watch(w, c)
	}()

	for {
		select {
		case <-w.Ctx().Done():
			logger.Error(w.Ctx().Err())
			w.Refresh()
			c = w.GetWatchChan()
			Mapping[w] = c
			goto Over
		case resp := <-c:
			if resp.Canceled {
				logger.Warnf("watch canceled")
				logger.Error(w.Ctx().Err())
				w.Refresh()
				c = w.GetWatchChan()
				Mapping[w] = c
				goto Over
			}
			table := w.GetTable()
			if len(resp.Events) > 0 {
				for _, evt := range resp.Events {
					switch evt.Type {
					case mvccpb.PUT:
						table.PushWatchEvent(routing.WatchMsg{
							Handle: func() routing.WatchMsgFunc {
								key := string(evt.Kv.Key)
								value := string(evt.Kv.Value)
								return func() {
									if err := w.Put(key, value, evt.IsCreate()); err != nil {
										logger.Error(err)
									}
								}
							}(),
						})
					case mvccpb.DELETE:
						table.PushWatchEvent(routing.WatchMsg{
							Handle: func() routing.WatchMsgFunc {
								key := string(evt.Kv.Key)
								return func() {
									if err := w.Delete(key); err != nil {
										logger.Error(err)
									}
								}
							}(),
						})
					default:
						logger.Warnf("unrecognized event type: %d", evt.Type)
						continue
					}
				}
			}
		}
	}

Over:
	logger.Debugf("watch task finished")
}

func Watch(wch map[Watcher]clientv3.WatchChan) {
	for k, v := range wch {
		if k.GetTable() == nil {
			panic("watcher does not bind to routing table")
		}
		go watch(k, v)
	}
}
