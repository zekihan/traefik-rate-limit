package rate_limit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/zekihan/traefik-rate-limit/internal/comm"
	"github.com/zekihan/traefik-rate-limit/internal/config"
)

var (
	redisClient redis.UniversalClient
	once        sync.Once
)

func getRedisClient() redis.UniversalClient {
	once.Do(func() {
		cfg := config.GetConfig()
		redisCfg := cfg.Redis
		redisClient = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:            redisCfg.Addrs,
			ClientName:       "TraefikRateLimiter",
			DB:               redisCfg.DB,
			Username:         redisCfg.Username,
			Password:         redisCfg.Password,
			SentinelUsername: redisCfg.SentinelUsername,
			SentinelPassword: redisCfg.SentinelPassword,
			PoolSize:         32, // Increased from 19 for better concurrency
			MasterName:       redisCfg.MasterName,
			MaxRetries:       3,
			MinRetryBackoff:  8 * time.Millisecond,
			MaxRetryBackoff:  512 * time.Millisecond,
		})
	})
	return redisClient
}

func RateLimit(data *comm.RateLimitRequestData) (*redis_rate.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // Reduced timeout
	defer cancel()
	limiter := redis_rate.NewLimiter(getRedisClient())
	result, err := limiter.Allow(ctx, data.Key, redis_rate.Limit{
		Rate:   int(data.Rate),
		Burst:  int(data.Burst),
		Period: data.Period,
	})
	if err != nil {
		return nil, fmt.Errorf("rate limit failed: %w", err)
	}
	return result, nil
}
