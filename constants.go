package traefik_rate_limit

const (
	XForwardedFor = "X-Forwarded-For"
)

type ContextKey string

const (
	RetryCountKey ContextKey = "retryCount"
)
