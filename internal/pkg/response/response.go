package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, status int, payload interface{}) {
	c.JSON(status, gin.H{"data": payload})
}

func Error(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func ErrorWithMessage(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}
