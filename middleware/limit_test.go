package middleware

import (
	"fmt"
	"testing"
	"time"
)

func TestLimiter_Burst(t *testing.T) {
	c := &ch
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
		fmt.Printf("burst time cost: %.3f μs\n", float64(time.Now().UnixNano() - t1) / 1e3)
		time.Sleep(1 *time.Second)
	}
}
