module github.com/zekihan/traefik-rate-limit

go 1.23.2

require (
	github.com/go-redis/redis_rate/v10 v10.0.1
	github.com/redis/go-redis/v9 v9.7.3
)

replace github.com/redis/go-redis/v9 v9.7.3 => ./redis
