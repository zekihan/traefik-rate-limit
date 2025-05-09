package traefik_rate_limit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Config the plugin configuration.
type Config struct {
	LogLevel          string            `json:"logLevel,omitempty"`
	Ratelimit         *RatelimitConfig  `json:"rateLimit,omitempty"`
	IPResolver        *IPResolverConfig `json:"ipResolver,omitempty"`
	WhitelistedIPNets []string          `json:"whitelistedIPNets,omitempty"`
	WhitelistLocalIPs bool              `json:"whitelistLocalIPs,omitempty"`
	SocketPath        string            `json:"socketPath,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		LogLevel: "info",
		Ratelimit: &RatelimitConfig{
			Rate:   100,
			Burst:  100,
			Period: time.Hour.String(),
		},
		IPResolver: &IPResolverConfig{
			Header:   "",
			UseSrcIP: true,
		},
		WhitelistedIPNets: make([]string, 0),
		WhitelistLocalIPs: true,
		SocketPath:        "",
	}
}

func (c *Config) Validate() error {
	if c.Ratelimit == nil {
		return fmt.Errorf("missing ratelimit configuration")
	}
	if err := c.Ratelimit.Validate(); err != nil {
		return fmt.Errorf("invalid ratelimit configuration")
	}
	return nil
}

// New created a new RateLimiter plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rateLimiter := &RateLimiter{
		next: next,
		name: name,
	}

	if config == nil {
		return rateLimiter, fmt.Errorf("missing configuration")
	}
	if err := config.Validate(); err != nil {
		return rateLimiter, err
	}
	period, err := time.ParseDuration(config.Ratelimit.Period)
	if err != nil {
		return nil, fmt.Errorf("invalid period: %v", err)
	}
	slog.Debug("Parsed period", slog.String("period", config.Ratelimit.Period), slog.Any("duration", period), slog.Any("error", err), slog.Any("type", reflect.TypeOf(period)))

	config.Ratelimit.period = period
	rateLimiter.conf = config

	logLevel := &slog.LevelVar{}
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	case "":
		logLevel.Set(slog.LevelInfo)
	default:
		slog.Warn("Invalid log level, using info", slog.String("level", config.LogLevel))
		logLevel.Set(slog.LevelInfo)
	}

	socketPath := config.SocketPath
	if socketPath == "" {
		socketPath = "/tmp/traefik-ratelimit.sock"
	}
	rateLimiter.socketPath = socketPath

	pluginLogger := NewPluginLogger(name, logLevel)
	rateLimiter.logger = pluginLogger

	rateLimiter.ipResolver = &IPResolver{
		config: config.IPResolver,
		logger: rateLimiter.logger,
	}

	whitelistedIPNets := make([]*net.IPNet, 0)
	if config.WhitelistLocalIPs {
		localIPs, err := rateLimiter.ipResolver.getLocalIPsHardcoded()
		if err != nil {
			return nil, fmt.Errorf("error getting local IPs: %v", err)
		}
		whitelistedIPNets = append(whitelistedIPNets, localIPs...)
	}
	for _, ipRange := range config.WhitelistedIPNets {
		_, ipNet, err := net.ParseCIDR(ipRange)
		if err != nil {
			return nil, fmt.Errorf("invalid whitelisted IP range: %s", ipRange)
		}
		whitelistedIPNets = append(whitelistedIPNets, ipNet)
	}
	rateLimiter.whitelistedIPNets = whitelistedIPNets

	return rateLimiter, nil
}

func (a *RateLimiter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer a.handlePanic(rw, req)

	ip, err := a.ipResolver.getIP(req)
	if err != nil {
		a.logger.Error("Error getting IP", ErrorAttrWithoutStack(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	a.logger.Debug("Request received", slog.String("ip", ip.String()), slog.String("method", req.Method), slog.String("path", req.URL.Path))

	if a.ipResolver.isWhitelisted(ip, a.whitelistedIPNets) {
		a.logger.Debug("IP is whitelisted, skipping rate limit", slog.String("ip", ip.String()))
		a.next.ServeHTTP(rw, req)
		return
	}

	ctx := req.Context()
	res, err := a.Allow(ctx, ip.String())
	if err != nil {
		a.logger.Error("Error getting rate limit", ErrorAttrWithoutStack(err))
		a.next.ServeHTTP(rw, req)
		return
	}
	a.logger.Debug("Rate limit response", slog.String("key", ip.String()), slog.Int64("allowed", res.Allowed), slog.Int64("remaining", res.Remaining), slog.Duration("resetAfter", res.ResetAfter))

	if res.Allowed <= 0 {
		retryAfter := int64(res.RetryAfter/time.Second) + 1
		rw.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
		http.Error(rw, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		return
	}

	a.next.ServeHTTP(rw, req)
}

func (a *RateLimiter) handlePanic(rw http.ResponseWriter, req *http.Request) {
	r := recover()
	err := getPanicError(r)
	if err == nil {
		return
	}

	if errors.Is(err, http.ErrAbortHandler) {
		return // suppress
	}

	a.logger.Error("Panic recovered", ErrorAttr(err))
	http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	//a.next.ServeHTTP(rw, req)
}

func getPanicError(r any) error {
	if r == nil {
		return nil
	}

	err, ok := r.(error)
	if ok {
		return err
	}

	refVal, ok := r.(reflect.Value)
	if ok && refVal.IsValid() && refVal.CanInterface() {
		refValInt := refVal.Interface()
		if err, ok := refValInt.(error); ok {
			return err
		}
	}

	return fmt.Errorf("panic: %v", r)
}
