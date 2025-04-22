package traefik_rate_limit

type ContextKey string

const (
	RetryCountKey ContextKey = "retryCount"
	XForwardedFor            = "X-Forwarded-For"
)
