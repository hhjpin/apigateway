package core

import (
	"github.com/satori/go.uuid"
	"time"
)

type RateLimitToken struct {
	u uuid.UUID
	seq int64
}

type TokenBucketRateLimit []RateLimitToken

func (t TokenBucketRateLimit) Len() int {return len(t)}

func (t TokenBucketRateLimit) Less(i, j int) bool {return t[i].seq < t[j].seq}

func (t TokenBucketRateLimit) Swap(i, j int) {t[i], t[j] = t[j], t[i]}

func (t *TokenBucketRateLimit) Push(x RateLimitToken) {*t = append(*t, x)}

func (t *TokenBucketRateLimit) Pop() RateLimitToken {
	old := *t
	n := len(old)
	x := old[n-1]
	*t = old[0:n-1]
	return x
}

func NewToken() RateLimitToken {
	uuid.NewV4()
	return RateLimitToken{
		u:
	} time.Now().UnixNano()
}
