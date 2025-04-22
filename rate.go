package traefik_rate_limit

import (
	"context"
	"fmt"
	"github.com/zekihan/traefik-rate-limit/internal/client"
	"github.com/zekihan/traefik-rate-limit/internal/comm"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

type RatelimitConfig struct {
	// Rate is the number of requests recovered per period.
	Rate int `json:"rate,omitempty"`

	// Burst is the maximum number of requests allowed in a burst.
	Burst int `json:"burst,omitempty"`

	// Period is the time period for the rate limit.
	Period string `json:"period,omitempty"`

	// period is the parsed time duration for the rate limit.
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
	socketPath        string
}

func (a *RateLimiter) GetKey(ip string) string {
	prefix := "traefik"
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

func (a *RateLimiter) Allow(ctx context.Context, ip string) (res *comm.RateLimitResponseData, err error) {
	if a.conf == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	if a.conf.Ratelimit == nil {
		return nil, fmt.Errorf("missing ratelimit configuration")
	}
	if ip == "" {
		return nil, fmt.Errorf("missing ip address")
	}

	limit := &comm.RateLimitRequestData{
		Rate:   uint64(a.conf.Ratelimit.Rate),
		Burst:  uint64(a.conf.Ratelimit.Burst),
		Period: a.conf.Ratelimit.period,
		Key:    a.GetKey(ip),
	}

	defer func() {
		if r := recover(); r != nil {
			a.logger.Debug("Recovered from panic", slog.Any("error", r))
			res = nil
			err = fmt.Errorf("%v", r)
		}
	}()
	newClient, err := client.NewClient(a.socketPath)
	if err != nil {
		panic(err)
	}
	defer newClient.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err = newClient.RateLimit(ctx, limit)
	if err != nil {
		slog.Info("failed to send request", slog.Any("error", err))
		return nil, err
	}
	newClient.Close()
	cancel()
	return res, nil
}
