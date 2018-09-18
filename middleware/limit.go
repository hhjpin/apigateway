package middleware

import (
	"git.henghajiang.com/backend/golang_utils/errors"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/valyala/fasthttp"
	"sync"
	"time"
)

type CountChan chan string

type limiter struct {
	limit uint64
	consume uint64
	interval int64

	sync.RWMutex
	internal map[string]uint64
}

var (
	Limiter = &limiter{
		limit:    3,
		consume:  1,
		interval: 5,
		internal: map[string]uint64{},
	}
	ch = make(CountChan, 1000)
	limitLogger = log.New()
)

func init() {
	limitLogger.EnableDebug()

	go Limiter.run()
	go Limiter.consuming()
}

func (l *limiter) Work(ctx *fasthttp.RequestCtx) error {
	remoteIP := ctx.RemoteIP().String()
	ch <- remoteIP
	burst := l.Burst(remoteIP)
	limitLogger.Debugf("frequency control info push into queue: %s", remoteIP)
	limitLogger.Debugf("remote ip request burst: %d, limit: %d", burst, l.limit)
	if burst >= l.limit {
		return errors.New(10)
	}
	return nil
}

func (l *limiter) run() {
	for rec := range ch{
		l.Lock()
		if c, ok := l.internal[rec]; ok {
			if c < l.limit {
				l.internal[rec] = c + 1
			}
		} else {
			l.internal[rec] = 1
		}
		l.Unlock()
	}
}

func (l *limiter) consuming() {
	for {
		l.Lock()
		for k, v := range l.internal {
			if v < l.consume {
				delete(l.internal, k)
			} else {
				l.internal[k] = v - l.consume
			}
		}
		l.Unlock()
		time.Sleep(time.Duration(l.interval) * time.Second)
	}
}

func (l *limiter) Limit() uint64 {
	return l.limit
}

func (l *limiter) Burst(input string) (burst uint64) {
	l.RLock()
	if b, ok := l.internal[input]; ok {
		burst = b
	} else {
		burst = 0
	}
	l.RUnlock()
	return
}