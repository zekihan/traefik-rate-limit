package config

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"log"
	"os"
	"sync"
)

const EnvKeyPrefix = "TRAEFIK_RATE_LIMIT__"

var configSyncOnce sync.Once
var config *Config

type RedisConfig struct {
	Addrs            []string `env:"ADDRS, default=localhost:6379"`
	DB               int      `env:"DB, default=0"`
	Username         string   `env:"USERNAME"`
	Password         string   `env:"PASSWORD"`
	SentinelUsername string   `env:"SENTINEL_USERNAME"`
	SentinelPassword string   `env:"SENTINEL_PASSWORD"`
	MasterName       string   `env:"MASTER_NAME"`
}

type Config struct {
	LogLevel   string       `env:"LOG_LEVEL, default=info"`
	SocketPath string       `env:"SOCKET_PATH, default=./tmp/traefik-rate-limit.sock"`
	Redis      *RedisConfig `env:", prefix=REDIS_"`
}

func newConfig() *Config {
	cfg := &Config{}
	ctx := context.Background()
	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   cfg,
		Lookuper: envconfig.PrefixLookuper(EnvKeyPrefix, envconfig.OsLookuper()),
	}); err != nil {
		log.Fatalf("Failed to process env config: %v", err)
	}
	return cfg
}

func newConfigWithEnv() *Config {
	// load env file
	envFile := ".env"
	if f, err := os.Stat(envFile); !(os.IsNotExist(err) || f.IsDir()) {
		err = godotenv.Load(envFile)
		if err != nil {
			panic(fmt.Sprintf("Error loading %s file", envFile))
		}
	}

	overrideEnvFile := ".override.env"
	if f, err := os.Stat(overrideEnvFile); !(os.IsNotExist(err) || f.IsDir()) {
		err = godotenv.Overload(overrideEnvFile)
		if err != nil {
			panic(fmt.Sprintf("Error loading %s file", overrideEnvFile))
		}
	}

	return newConfig()
}

func GetConfig() *Config {
	configSyncOnce.Do(func() {
		config = newConfigWithEnv()
	})
	return config
}
