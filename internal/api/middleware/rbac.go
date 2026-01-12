package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	userSvc "github.com/open-apime/apime/internal/service/user"
)

// RequireAdmin cria um middleware que exige que o usuário seja admin
func RequireAdmin(userService *userSvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
			return
		}

		user, err := userService.Get(c.Request.Context(), userID.(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não encontrado"})
			return
		}

		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acesso negado: apenas administradores"})
			return
		}

		// Adiciona informações do usuário ao contexto
		c.Set("userRole", user.Role)
		c.Next()
	}
}

// RequireRole cria um middleware que exige que o usuário tenha um role específico
func RequireRole(userService *userSvc.Service, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
			return
		}

		user, err := userService.Get(c.Request.Context(), userID.(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não encontrado"})
			return
		}

		if user.Role != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acesso negado: permissão insuficiente"})
			return
		}

		// Adiciona informações do usuário ao contexto
		c.Set("userRole", user.Role)
		c.Next()
	}
}

// AddUserInfo adiciona informações do usuário ao contexto para endpoints que precisam
func AddUserInfo(userService *userSvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.Next()
			return
		}

		user, err := userService.Get(c.Request.Context(), userID.(string))
		if err != nil {
			// Se não encontrar o usuário, continua sem adicionar info
			c.Next()
			return
		}

		// Adiciona informações do usuário ao contexto
		c.Set("userRole", user.Role)
		c.Set("userEmail", user.Email)
		c.Next()
	}
}
