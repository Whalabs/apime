package middleware

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/sentryx"
)

// SentryReport reports to Sentry any errors accumulated via c.Error() and any
// 5xx responses, tagged with the route and request-id. It only runs when Sentry
// is enabled (conditional registration in the router).
func SentryReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		tags := map[string]string{
			"method": c.Request.Method,
			"route":  routeOrPath(c),
		}
		if rid := c.GetString(HeaderRequestID); rid != "" {
			tags["request_id"] = rid
		}

		route := routeOrPath(c)
		reported := 0
		for _, ginErr := range c.Errors {
			// Self-recovering failures (socket down mid-send, IQ without answer): the request
			// already answered 503 and the caller retries, so reporting here is pure noise. Same
			// criterion applied to the whatsmeow logger.
			if sentryx.IsExpectedNoise(ginErr.Err.Error()) {
				continue
			}
			// The message carries phone/JID/ID, so without an explicit fingerprint each occurrence
			// abriria uma issue nova. Agrupa por rota + esqueleto do texto.
			sentryx.CaptureErrorWithFingerprint(ginErr.Err, tags,
				sentryx.Fingerprint("http:"+route, ginErr.Err.Error()))
			reported++
		}

		// The generic one only fires when NOTHING was reported above: with an error already
		// recorded it would be a second event for the same problem, saying less.
		if c.Writer.Status() >= 500 && reported == 0 && len(c.Errors) == 0 {
			sentryx.CaptureMessageWithFingerprint(
				fmt.Sprintf("HTTP %d %s %s", c.Writer.Status(), c.Request.Method, route),
				sentry.LevelError,
				tags,
				[]string{"http:" + route, fmt.Sprintf("status:%d", c.Writer.Status())},
			)
		}
	}
}

func routeOrPath(c *gin.Context) string {
	if fp := c.FullPath(); fp != "" {
		return fp
	}
	return c.Request.URL.Path
}
