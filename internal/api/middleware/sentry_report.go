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
			// Falha que a stack se recupera sozinha (socket caído no meio de um envio, IQ sem
			// resposta): a requisição já respondeu 503 e o chamador reenvia. Reportar aqui só
			// gera ruído, e é o mesmo critério aplicado ao logger do whatsmeow.
			if sentryx.IsExpectedNoise(ginErr.Err.Error()) {
				continue
			}
			// A mensagem carrega telefone/JID/ID, então sem fingerprint próprio cada ocorrência
			// abriria uma issue nova. Agrupa por rota + esqueleto do texto.
			sentryx.CaptureErrorWithFingerprint(ginErr.Err, tags,
				sentryx.Fingerprint("http:"+route, ginErr.Err.Error()))
			reported++
		}

		// O genérico só entra quando NADA foi reportado acima: com um erro registrado ele seria um
		// segundo evento para o mesmo problema, dizendo menos (só o status, sem a causa).
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
