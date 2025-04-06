package traefik_rate_limit

import (
	"context"
	"fmt"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

type RatelimitConfig struct {
	Rate   int    `json:"rate,omitempty"`
	Burst  int    `json:"burst,omitempty"`
	Period string `json:"period,omitempty"`
	period time.Duration
}

func (c *RatelimitConfig) Validate() error {
	if c.Rate <= 0 {
		return fmt.Errorf("rate must be greater than 0")
	}
	if c.Burst <= 0 {
		return fmt.Errorf("burst must be greater than 0")
	}
	period, err := time.ParseDuration(c.Period)
	if err != nil {
		return fmt.Errorf("invalid period: %v", err)
	}
	if period <= time.Duration(0) {
		return fmt.Errorf("period must be greater than 0")
	}
	c.period = period
	return nil
}

// RateLimiter plugin.
type RateLimiter struct {
	next              http.Handler
	name              string
	conf              *Config
	logger            *PluginLogger
	ipResolver        *IPResolver
	whitelistedIPNets []*net.IPNet
}

func (a *RateLimiter) GetKey(ip string) string {
	prefix := a.conf.Redis.Prefix
	if prefix == "" {
		prefix = "traefik"
	}
	name := url.PathEscape(a.name)
	if name == "" {
		name = "default"
	}

	ipKey := url.PathEscape(ip)
	if ipKey == "" {
		ipKey = "default"
	}

	key := fmt.Sprintf("%s:%s:%s", prefix, name, ipKey)
	return key
}

func (a *RateLimiter) Allow(ctx context.Context, ip string) (res *redis_rate.Result, err error) {
	if a.conf == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	if a.conf.Ratelimit == nil {
		return nil, fmt.Errorf("missing ratelimit configuration")
	}
	if a.conf.Redis == nil {
		return nil, fmt.Errorf("missing redis configuration")
	}
	if ip == "" {
		return nil, fmt.Errorf("missing ip address")
	}

	limit := redis_rate.Limit{
		Rate:   a.conf.Ratelimit.Rate,
		Burst:  a.conf.Ratelimit.Burst,
		Period: a.conf.Ratelimit.period,
	}

	defer func() {
		if r := recover(); r != nil {
			a.logger.Debug("Recovered from panic", slog.Any("error", r))
			res = nil
			err = fmt.Errorf("%v", r)
		}
	}()
	rdb := redis.GetClient(&redis.Options{
		Addr: a.conf.Redis.GetAddress(),
	})

	limiter := redis_rate.NewLimiter(rdb)
	return limiter.Allow(ctx, a.GetKey(ip), limit)
}
