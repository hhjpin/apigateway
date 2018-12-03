package middleware

import (
	"context"
	config "git.henghajiang.com/backend/api_gateway_v2/conf"
	"git.henghajiang.com/backend/golang_utils/errors"
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
	ReceiveChan []CountChan
}

type limiter struct {
	limit    uint64
	consume  uint64
	interval int64

	sync.RWMutex
	internal  map[string]uint64
	blackList map[string]*blackListItem
}

var (
	limiters Limiters
	shardNumber int
)

func init() {
	limiterConf := config.ReadConfig().Middleware.Limiter

	shardNumber = runtime.NumCPU()
	limiters = Limiters{
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
		limiters.limiterArray[i] = &limiter{
			limit:     limiterConf.DefaultLimit,
			consume:   limiterConf.DefaultConsumePerPeriod,
			interval:  limiterConf.DefaultConsumePeriod,
			blackList: blackList,
		}
		limiters.ReceiveChan[i] = make(CountChan, limiterConf.LimiterChanLength)
		go limiters.limiterArray[i].run(ctx, limiters.ReceiveChan[i])
		go limiters.limiterArray[i].consuming(ctx)
	}

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<- c
	cancel()
}

func (l *limiter) SetBlackList(ip string, limit uint64, expiresAt int64) {
	l.Lock()
	if b, ok := l.blackList[ip]; ok {
		b.limit = limit
		b.expiresAt = expiresAt
	} else {
		l.blackList[ip] = &blackListItem {
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

func (l *Limiters) SetBlackList(ip string, limit uint64, expiresAt int64){
	for _, i := range l.limiterArray {
		i.SetBlackList(ip, limit, expiresAt)
	}
}

func (l *Limiters) Work(ctx *fasthttp.RequestCtx, errChan chan error) {
	remoteIP := ctx.RemoteIP().String()
	shardInt := murmur.Sum32(remoteIP)
	shard := int(shardInt) % shardNumber
	l.ReceiveChan[shard] <- remoteIP

	limiter := l.limiterArray[shard]
	burst := limiter.Burst(remoteIP)
	black, exists := limiter.VerifyBlackList(remoteIP)
	if exists {
		logger.Debugf("ip %s in blacklist, limit: %d", remoteIP, black)
		if burst >= black {
			errChan <- errors.NewFormat(11, "请联系客服解除封禁限制")
		}
	} else {
		logger.Debugf("remote ip request burst: %d, limit: %d", burst, limiter.limit)
		if burst >= limiter.limit {
			errChan <- errors.New(10)
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
			if c < l.limit {
				l.internal[rec] = c + 1
			}
		} else {
			l.internal[rec] = 1
		}
		l.Unlock()
	}
}

func (l *limiter) consuming(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// pass
		}
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
