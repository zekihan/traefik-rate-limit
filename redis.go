package traefik_rate_limit

import (
	"fmt"
)

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
