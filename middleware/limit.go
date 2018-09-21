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
	limit    uint64
	consume  uint64
	interval int64

	sync.RWMutex
	internal  map[string]uint64
	blackList map[string]*struct {
		limit     uint64
		expiresAt int64
	}
}

var (
	Limiter = &limiter{
		limit:    3,
		consume:  1,
		interval: 5,
		internal: map[string]uint64{},
		blackList: map[string]*struct {
			limit     uint64
			expiresAt int64
		}{},
	}
	limitCh     = make(CountChan, 1000)
	limitLogger = log.New()
)

func init() {
	limitLogger.EnableDebug()

	go Limiter.run()
	go Limiter.consuming()
}

func (l *limiter) SetBlackList(ip string, limit uint64, expiresAt int64) {
	l.Lock()
	if b, ok := l.blackList[ip]; ok {
		b.limit = limit
		b.expiresAt = expiresAt
	} else {
		l.blackList[ip] = &struct {
			limit     uint64
			expiresAt int64
		}{
			limit:     limit,
			expiresAt: expiresAt,
		}
	}
	l.Unlock()
}

func (l *limiter) VerifyBlackList(ip string) (uint64, bool) {
	l.RLock()
	defer l.RUnlock()
	b, ok := l.blackList[ip]
	if ok {
		if time.Now().Unix() < b.expiresAt {
			return b.limit, true
		} else {
			return 0, false
		}
	} else {
		return 0, false
	}
}

func (l *limiter) Work(ctx *fasthttp.RequestCtx, errChan chan error) {
	remoteIP := ctx.RemoteIP().String()
	limitCh <- remoteIP
	burst := l.Burst(remoteIP)
	black, exists := l.VerifyBlackList(remoteIP)
	if exists {
		limitLogger.Debugf("ip %s in blacklist, limit: %d", remoteIP, black)
		if burst >= black {
			errChan <- errors.NewFormat(11, "请联系客服解除封禁限制")
		}
	} else {
		limitLogger.Debugf("remote ip request burst: %d, limit: %d", burst, l.limit)
		if burst >= l.limit {
			errChan <- errors.New(10)
		} else {
			errChan <- nil
		}
	}
	return
}

func (l *limiter) run() {
	for rec := range limitCh {
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
		for k, v := range l.blackList {
			if time.Now().Unix() > v.expiresAt {
				delete(l.blackList, k)
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
