package middleware

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/sentryx"
)

// SentryReport reporta ao Sentry quaisquer erros acumulados via c.Error() e
// respostas 5xx, com tags da rota e do request-id. Roda só quando o Sentry
// estiver habilitado (registro condicional no router).
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

		for _, ginErr := range c.Errors {
			sentryx.CaptureError(ginErr.Err, tags)
		}

		if c.Writer.Status() >= 500 {
			sentryx.CaptureMessage(
				fmt.Sprintf("HTTP %d %s %s", c.Writer.Status(), c.Request.Method, routeOrPath(c)),
				sentry.LevelError,
				tags,
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
