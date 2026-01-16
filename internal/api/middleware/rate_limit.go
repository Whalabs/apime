package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var rateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {current, ttl}
`)

// RateLimitOption parametriza o middleware de limite por token.
type RateLimitOption struct {
	Enabled     bool
	Requests    int
	Window      time.Duration
	Prefix      string
	RedisClient redis.Scripter
	Logger      *zap.Logger
}

// RateLimit aplica contagem de requisições por token usando Redis.
func RateLimit(opts RateLimitOption) gin.HandlerFunc {
	if !opts.Enabled || opts.RedisClient == nil || opts.Requests <= 0 || opts.Window <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	prefix := opts.Prefix
	if prefix == "" {
		prefix = "ratelimit:api"
	}
	windowMs := opts.Window.Milliseconds()

	return func(c *gin.Context) {
		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("%s:%s", prefix, hashToken(token))
		vals, err := rateLimitScript.Run(c.Request.Context(), opts.RedisClient, []string{key}, windowMs).Int64Slice()
		if err != nil {
			if opts.Logger != nil {
				opts.Logger.Warn("rate limit: erro ao consultar redis", zap.Error(err))
			}
			c.Next()
			return
		}
		if len(vals) < 2 {
			c.Next()
			return
		}

		current := vals[0]
		ttlMs := vals[1]
		remaining := opts.Requests - int(current)
		if remaining < 0 {
			remaining = 0
		}
		resetAfter := durationFromTTL(ttlMs, opts.Window)
		resetUnix := time.Now().Add(resetAfter).Unix()

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", opts.Requests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))

		if current > int64(opts.Requests) {
			retryAfter := durationFromTTL(ttlMs, opts.Window)
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "limite de requisições excedido",
			})
			return
		}

		c.Next()
	}
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func durationFromTTL(ttlMs int64, fallback time.Duration) time.Duration {
	if ttlMs > 0 {
		return time.Duration(ttlMs) * time.Millisecond
	}
	return fallback
}
