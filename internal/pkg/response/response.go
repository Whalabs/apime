package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, status int, payload interface{}) {
	c.JSON(status, gin.H{"data": payload})
}

func Error(c *gin.Context, status int, err error) {
	// Register the error in the gin context so the Sentry middleware
	// (SentryReport) captures the real error via CaptureError, instead of just
	// the generic "HTTP <status>" message.
	if err != nil {
		_ = c.Error(err)
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func ErrorWithMessage(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}

// InstanceNotConnected carries a stable code next to the message, so clients branch on it instead
// of matching the Portuguese text.
func InstanceNotConnected(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "instância não conectada",
		"code":  "instance_not_connected",
	})
}
