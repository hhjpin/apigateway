package core

import (
	"context"
	"github.com/golang/time/rate"
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) {
	l := rate.NewLimiter(100, 200)
	c, _ := context.WithCancel(context.TODO())
	for {
		l.Wait(c)
		time.Sleep(10000 * time.Nanosecond)
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"))
	}
	l.Limit()
}
