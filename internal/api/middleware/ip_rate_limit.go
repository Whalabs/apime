package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var ipRateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {current, ttl}
`)

type IPRateLimitOption struct {
	Enabled        bool
	Requests       int
	WindowSeconds  int
	RedisClient    redis.Scripter
	Logger         *zap.Logger
	SkipPrivateIPs bool
}

// IPRateLimit reforça limites por IP para rotas públicas.
func IPRateLimit(opts IPRateLimitOption) gin.HandlerFunc {
	if !opts.Enabled || opts.RedisClient == nil || opts.Requests <= 0 || opts.WindowSeconds <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	windowMs := int64(opts.WindowSeconds) * 1000

	return func(c *gin.Context) {
		clientIP := GetClientIP(c)

		if opts.SkipPrivateIPs && IsPrivateIP(clientIP) {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:ip:%s", hashIP(clientIP))

		vals, err := ipRateLimitScript.Run(c.Request.Context(), opts.RedisClient, []string{key}, windowMs).Int64Slice()
		if err != nil {
			if opts.Logger != nil {
				opts.Logger.Warn("ip rate limit: erro ao consultar redis", zap.Error(err))
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

		resetAfter := durationFromTTL(ttlMs, time.Duration(opts.WindowSeconds)*time.Second)
		resetUnix := time.Now().Add(resetAfter).Unix()

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", opts.Requests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))

		if current > int64(opts.Requests) {
			retryAfter := durationFromTTL(ttlMs, time.Duration(opts.WindowSeconds)*time.Second)
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

			if opts.Logger != nil {
				opts.Logger.Warn("ip rate limit: limite excedido",
					zap.String("ip", clientIP),
					zap.Int64("attempts", current),
				)
			}

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "muitas tentativas. tente novamente mais tarde",
			})
			return
		}

		c.Next()
	}
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}
