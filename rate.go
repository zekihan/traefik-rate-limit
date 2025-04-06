package traefik_rate_limit

import (
	"context"
	"fmt"
	"github.com/go-redis/redis_rate/v10"
	"net"
	"net/http"
	"net/url"
	"time"
)

type RatelimitConfig struct {
	Rate   int           `json:"rate,omitempty"`
	Burst  int           `json:"burst,omitempty"`
	Period time.Duration `json:"period,omitempty"`
}

func (c *RatelimitConfig) Validate() error {
	if c.Rate <= 0 {
		return fmt.Errorf("rate must be greater than 0")
	}
	if c.Burst <= 0 {
		return fmt.Errorf("burst must be greater than 0")
	}
	if c.Period <= 0 {
		return fmt.Errorf("period must be greater than 0")
	}
	return nil
}

// RateLimiter plugin.
type RateLimiter struct {
	next              http.Handler
	name              string
	conf              *Config
	logger            *PluginLogger
	limiter           *redis_rate.Limiter
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

func (a *RateLimiter) Allow(ctx context.Context, ip string) (*redis_rate.Result, error) {
	if a.limiter == nil {
		return nil, fmt.Errorf("missing redis rate limiter")
	}
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
		Period: a.conf.Ratelimit.Period,
	}
	return a.limiter.Allow(ctx, a.GetKey(ip), limit)
}
