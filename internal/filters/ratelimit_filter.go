package filters

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

const MICROSECS_PER_SEC = 1_000_000

type TokenBucket struct {
	tokens     int
	rate       int
	tokensLock sync.Mutex
	lastCheck  time.Time
}

func (t *TokenBucket) AddNewTokens() {
	t.tokensLock.Lock()
	defer t.tokensLock.Unlock()

	uSecsSinceLastCheck := time.Since(t.lastCheck).Microseconds()
	newTokens := uSecsSinceLastCheck * int64(t.rate) / MICROSECS_PER_SEC

	if newTokens > 0 {
		t.tokens += int(newTokens)

		if t.tokens > t.rate {
			t.tokens = t.rate
		}

		t.lastCheck = time.Now()
	}
}

func (t *TokenBucket) TryConsumeTokens(numTokens int) (withinRatelimit bool) {
	t.tokensLock.Lock()
	defer t.tokensLock.Unlock()

	if t.tokens >= numTokens {
		t.tokens -= numTokens
		return true
	}

	return false
}

func NewTokenBucket(rate int, startFull bool) *TokenBucket {
	tokens := 0
	if startFull {
		tokens = rate
	}

	return &TokenBucket{
		tokens:     tokens,
		rate:       rate,
		tokensLock: sync.Mutex{},
		lastCheck:  time.Now(),
	}
}

type RateLimitFilterConfig struct {
	PartitionKey string
	Rate         int
}

func NewRateLimitFilterConfigFromRaw(raw map[string]string) (RateLimitFilterConfig, error) {
	var partitionKey string
	if key, ok := raw["partition_key"]; ok {
		partitionKey = key
	} else {
		return RateLimitFilterConfig{}, fmt.Errorf("missing `partition_key` in RateLimitFilter")
	}

	var rate int
	if rateStr, ok := raw["rate"]; ok {
		if val, err := strconv.Atoi(rateStr); err == nil {
			if val <= 0 {
				return RateLimitFilterConfig{}, fmt.Errorf("invalid rate in RateLimitFilter - expected a positive int, got %d", val)
			}
			rate = val
		} else {
			return RateLimitFilterConfig{}, fmt.Errorf("invalid rate in RateLimitFilter - expected an int, got `%s`", rateStr)
		}
	} else {
		return RateLimitFilterConfig{}, fmt.Errorf("missing `rate` in RateLimitFilter")
	}

	return RateLimitFilterConfig{
		Rate:         rate,
		PartitionKey: partitionKey,
	}, nil
}

type RateLimitFilter struct {
	RateLimitFilterConfig
	buckets     map[string]*TokenBucket
	bucketsLock sync.Mutex
}

func NewRateLimitFilter(conf RateLimitFilterConfig) *RateLimitFilter {
	return &RateLimitFilter{
		RateLimitFilterConfig: conf,
		buckets:               make(map[string]*TokenBucket),
		bucketsLock:           sync.Mutex{},
	}
}

func (r *RateLimitFilter) Filter(ctx context.Context, msg *clogger.Message) (shouldDrop bool, err error) {
	key := fmt.Sprint(msg.ParsedFields[r.PartitionKey])
	r.bucketsLock.Lock()
	defer r.bucketsLock.Unlock()

	tokenBucket := r.buckets[key]
	if tokenBucket == nil {
		r.buckets[key] = NewTokenBucket(r.Rate, true)
		tokenBucket = r.buckets[key]
	}
	tokenBucket.AddNewTokens()
	hadTokenForMsg := tokenBucket.TryConsumeTokens(1)

	return !hadTokenForMsg, nil
}

func init() {
	filtersRegistry.Register("ratelimit", func(rawConf map[string]string) (interface{}, error) {
		return NewRateLimitFilterConfigFromRaw(rawConf)
	}, func(rawConf interface{}) (Filter, error) {
		if conf, ok := rawConf.(RateLimitFilterConfig); ok {
			return NewRateLimitFilter(conf), nil
		} else {
			return nil, fmt.Errorf("BUG: invalid type for RateLimit filter configuration (expected RateLimitFilterConfig)")
		}
	})
}
