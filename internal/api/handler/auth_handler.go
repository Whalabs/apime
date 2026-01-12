package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/response"
	authSvc "github.com/open-apime/apime/internal/service/auth"
)

type AuthHandler struct {
	service *authSvc.Service
}

func NewAuthHandler(service *authSvc.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(r *gin.RouterGroup) {
	r.POST("/auth/login", h.login)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"token": token})
}
