package traefik_rate_limit

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

func init() {
	ctx := context.Background()
	rdb := redis.GetClient(&redis.Options{
		Addr: "redis:6379",
	})
	_ = rdb.Ping(ctx).Err()
}

type RedisConfig struct {
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

func (c *RedisConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("missing redis host")
	}
	if c.Port <= 0 {
		return fmt.Errorf("invalid redis port")
	}
	if c.Prefix == "" {
		return fmt.Errorf("missing redis prefix")
	}
	return nil
}
