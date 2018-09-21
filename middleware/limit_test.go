package middleware

import (
	"fmt"
	"github.com/go-ego/murmur"
	"testing"
	"time"
)

func TestLimiter_Burst(t *testing.T) {
	c := &limitCh
	go func() {
		for {
			*c <- "192.168.1.16"
			*c <- "192.168.1.16"
			*c <- "192.168.1.16"
			*c <- "192.168.1.16"
			*c <- "192.168.1.16"
			time.Sleep(200 * time.Millisecond)
		}
	}()
	for {
		t1 := time.Now().UnixNano()
		fmt.Println(Limiter.Burst("192.168.1.16"))
		fmt.Printf("burst time cost: %.3f μs\n", float64(time.Now().UnixNano()-t1)/1e3)
		time.Sleep(1 * time.Second)
	}
}

func TestNewCounting(t *testing.T) {
	m := murmur.Sum32("1992141231331")
	s := uint32(4)
	limitLogger.Info(m)
	limitLogger.Info(m % s)
}
