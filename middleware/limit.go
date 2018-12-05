package middleware

import (
	"context"
	"fmt"
	config "git.henghajiang.com/backend/api_gateway_v2/conf"
	"git.henghajiang.com/backend/golang_utils/errors"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/go-ego/murmur"
	"github.com/valyala/fasthttp"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type CountChan chan string

type blackListItem struct {
	limit     uint64
	expiresAt int64
}

type Limiters struct {
	limiterArray []*limiter
	ReceiveChan  []CountChan
}

type limiter struct {
	limit    uint64
	consume  uint64
	interval int64

	sync.RWMutex
	internal  map[string]uint64
	blackList map[string]*blackListItem
}

const (
	maxUint64 uint64 = 18446744073709551614
	maxBannedCount uint64 = 10
)

var (
	Limiter     Limiters
	shardNumber int
)

func init() {
	limiterConf := config.ReadConfig().Middleware.Limiter

	shardNumber = runtime.NumCPU()
	Limiter = Limiters{
		limiterArray: make([]*limiter, shardNumber),
		ReceiveChan:  make([]CountChan, shardNumber),
	}

	blackList := make(map[string]*blackListItem)
	for idx, item := range limiterConf.DefaultBlackList {
		tmp := limiterConf.DefaultBlackList[idx]
		blackList[item.IP] = &blackListItem{
			limit:     tmp.Limit,
			expiresAt: tmp.ExpiresAt,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i <= shardNumber-1; i++ {
		Limiter.limiterArray[i] = &limiter{
			limit:     limiterConf.DefaultLimit,
			consume:   limiterConf.DefaultConsumePerPeriod,
			interval:  limiterConf.DefaultConsumePeriod,
			internal: make(map[string]uint64),
			blackList: blackList,
		}
		Limiter.ReceiveChan[i] = make(CountChan, limiterConf.LimiterChanLength)
		go Limiter.limiterArray[i].run(ctx, Limiter.ReceiveChan[i])
		go Limiter.limiterArray[i].consuming(ctx)
	}

	go func(cancelFunc context.CancelFunc) {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-c
		cancelFunc()
	}(cancel)
}

func (l *limiter) SetBlackList(ip string, limit uint64, expiresAt int64) {
	l.Lock()
	if b, ok := l.blackList[ip]; ok {
		b.limit = limit
		b.expiresAt = expiresAt
	} else {
		l.blackList[ip] = &blackListItem{
			limit:     limit,
			expiresAt: expiresAt,
		}
	}
	l.Unlock()
}

func (l *limiter) VerifyBlackList(ip string) (black *blackListItem, ok bool) {
	var flag bool
	l.RLock()
	b, ok := l.blackList[ip]
	if ok {
		if time.Now().Unix() < b.expiresAt {
			black = b
			ok = true
		} else {
			black = nil
			ok = false
			flag = true
		}
	} else {
		black = nil
		ok = false
	}
	l.RUnlock()
	if flag {
		l.Lock()
		delete(l.blackList, ip)
		l.Unlock()
	}
	return
}

func (l Limiters) SetBlackList(ip string, limit uint64, expiresAt int64) {
	for _, i := range l.limiterArray {
		i.SetBlackList(ip, limit, expiresAt)
	}
}

func (l Limiters) Work(ctx *fasthttp.RequestCtx, errChan chan error) {
	remoteIP := ctx.RemoteIP().String()
	shardInt := murmur.Sum32(remoteIP)
	shard := int(shardInt) % shardNumber
	logger.Debugf("request murmur: %d, shard: %d", shardInt, shard)
	l.ReceiveChan[shard] <- remoteIP

	limiter := l.limiterArray[shard]
	burst := limiter.Burst(remoteIP)
	black, exists := limiter.VerifyBlackList(remoteIP)
	if exists {
		logger.Debugf("ip %s in blacklist, limit: %d", remoteIP, black)
		if burst >= black.limit {
			errChan <- errors.NewFormat(11, fmt.Sprintf("您的访问过于频繁, 将于%s解除限制", time.Unix(black.expiresAt, 0).Format("2006-1-2 15:04:05")))
		}
	} else {
		logger.Debugf("remote ip request burst: %d, limit: %d", burst, limiter.limit)
		if burst >= limiter.limit && burst < maxBannedCount {
			errChan <- errors.New(10)
		} else if burst >= maxBannedCount {
			expires := time.Now().Unix() + 86400
			l.SetBlackList(remoteIP, 0, expires)
			errChan <- errors.NewFormat(11, fmt.Sprintf("您的访问过于频繁, 将于%s解除限制", time.Unix(expires, 0).Format("2006-1-2 15:04:05")))
		} else {
			errChan <- nil
		}
	}
	return
}

func (l *limiter) run(ctx context.Context, recv CountChan) {
	for rec := range recv {
		select {
		case <-ctx.Done():
			return
		default:
			// pass
		}
		l.Lock()
		if c, ok := l.internal[rec]; ok {
			if c < maxUint64 {
				l.internal[rec] = c + 1
			}
		} else {
			l.internal[rec] = 1
		}
		l.Unlock()
	}
}

func (l *limiter) consuming(ctx context.Context) {
	cLogger := log.New()
	cLogger.EnableDebug()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// pass
		}
		l.Lock()
		for k, v := range l.internal {
			cLogger.Debugf("limiter: %+v", *l)
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
