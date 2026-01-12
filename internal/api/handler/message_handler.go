package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/response"
	messageSvc "github.com/open-apime/apime/internal/service/message"
)

// normalizeJID normaliza um JID, adicionando @s.whatsapp.net apenas para números de telefone
// Grupos devem ser passados com JID completo (ex: 120363123456789012@g.us)
// Para números brasileiros, remove o 9º dígito (9) de números de celular se presente
func normalizeJID(jidStr string) string {
	jidStr = strings.TrimSpace(jidStr)

	// Se for grupo ou outro tipo que não seja @s.whatsapp.net, retorna como está
	if strings.Contains(jidStr, "@") {
		// Se for @s.whatsapp.net, extrai o número e normaliza
		if strings.HasSuffix(jidStr, "@s.whatsapp.net") {
			phone := strings.TrimSuffix(jidStr, "@s.whatsapp.net")
			normalized := normalizeBrazilianPhone(phone)
			return normalized + "@s.whatsapp.net"
		}
		// Outros tipos (@g.us, etc.) retorna como está
		return jidStr
	}

	// Normalizar número brasileiro: remover o 9º dígito de celulares
	normalized := normalizeBrazilianPhone(jidStr)

	return normalized + "@s.whatsapp.net"
}

// normalizeBrazilianPhone normaliza números de telefone brasileiros
// Remove o 9º dígito (9) de números de celular se presente
// Formato esperado: 55 + DDD (2 dígitos) + número (8 ou 9 dígitos)
// Exemplo: 5511999999999 -> 551199999999 (remove o 9º dígito)
func normalizeBrazilianPhone(phone string) string {
	// Remove caracteres não numéricos
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Verifica se é número brasileiro (começa com 55)
	if len(digits) < 4 || !strings.HasPrefix(digits, "55") {
		return phone // Não é brasileiro, retorna original
	}

	// Verifica se tem pelo menos DDD + 9 dígitos (55 + 2 + 9 = 13 dígitos)
	if len(digits) < 13 {
		return phone // Muito curto, retorna original
	}

	// Extrai: 55 (país) + DDD (2 dígitos) + número (resto)
	// Exemplo: 5511999999999 -> país=55, ddd=11, numero=999999999
	country := digits[0:2]  // 55
	areaCode := digits[2:4] // DDD (11-99)
	number := digits[4:]    // Número restante

	// Se o número tem 9 dígitos e começa com 9, remove o primeiro 9
	// Isso normaliza para o formato antigo que o WhatsApp pode esperar
	if len(number) == 9 && number[0] == '9' {
		// Remove o primeiro dígito (9)
		normalizedNumber := number[1:]
		return country + areaCode + normalizedNumber
	}

	return phone // Não precisa normalizar
}

type MessageHandler struct {
	service *messageSvc.Service
}

func NewMessageHandler(service *messageSvc.Service) *MessageHandler {
	return &MessageHandler{service: service}
}

func (h *MessageHandler) Register(r *gin.RouterGroup) {
	r.POST("/instances/:id/messages", h.enqueue)
	r.POST("/instances/:id/messages/text", h.sendText)
	r.POST("/instances/:id/messages/media", h.sendMedia)
	r.POST("/instances/:id/messages/audio", h.sendAudio)
	r.POST("/instances/:id/messages/document", h.sendDocument)
	r.GET("/instances/:id/messages", h.list)
}

type messageRequest struct {
	To      string `json:"to" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Payload string `json:"payload" binding:"required"`
}

func (h *MessageHandler) enqueue(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return
	}
	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	msg, err := h.service.Enqueue(c.Request.Context(), messageSvc.EnqueueInput{
		InstanceID: instanceID,
		To:         req.To,
		Type:       req.Type,
		Payload:    req.Payload,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusAccepted, msg)
}

type sendTextRequest struct {
	To   string `json:"to" binding:"required"`
	Text string `json:"text" binding:"required"`
}

func (h *MessageHandler) sendText(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return
	}
	var req sendTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	// Normalizar JID (adiciona @s.whatsapp.net se for apenas número)
	normalizedTo := normalizeJID(req.To)

	msg, err := h.service.Send(c.Request.Context(), messageSvc.SendInput{
		InstanceID: instanceID,
		To:         normalizedTo,
		Type:       "text",
		Text:       req.Text,
	})
	if err != nil {
		if errors.Is(err, messageSvc.ErrInstanceNotConnected) {
			response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		} else if errors.Is(err, messageSvc.ErrInvalidJID) {
			response.Error(c, http.StatusBadRequest, err)
		} else {
			response.Error(c, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(c, http.StatusOK, msg)
}

func (h *MessageHandler) sendMedia(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return
	}
	to := c.PostForm("to")
	mediaType := c.PostForm("type") // "image" ou "video"
	caption := c.PostForm("caption")

	if to == "" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "campo 'to' é obrigatório")
		return
	}

	if mediaType != "image" && mediaType != "video" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "tipo deve ser 'image' ou 'video'")
		return
	}

	// Obter arquivo
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "arquivo não fornecido")
		return
	}

	// Abrir arquivo
	src, err := file.Open()
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao abrir arquivo")
		return
	}
	defer src.Close()

	// Ler dados do arquivo
	fileData, err := io.ReadAll(src)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao ler arquivo")
		return
	}

	// Normalizar JID (adiciona @s.whatsapp.net se for apenas número)
	normalizedTo := normalizeJID(to)

	msg, err := h.service.Send(c.Request.Context(), messageSvc.SendInput{
		InstanceID: instanceID,
		To:         normalizedTo,
		Type:       mediaType,
		MediaData:  fileData,
		MediaType:  file.Header.Get("Content-Type"),
		Caption:    caption,
	})
	if err != nil {
		if errors.Is(err, messageSvc.ErrInstanceNotConnected) {
			response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		} else {
			response.Error(c, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(c, http.StatusOK, msg)
}

func (h *MessageHandler) sendAudio(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return
	}
	to := c.PostForm("to")

	if to == "" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "campo 'to' é obrigatório")
		return
	}

	// Obter arquivo
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "arquivo não fornecido")
		return
	}

	// Abrir arquivo
	src, err := file.Open()
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao abrir arquivo")
		return
	}
	defer src.Close()

	// Ler dados do arquivo
	fileData, err := io.ReadAll(src)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao ler arquivo")
		return
	}

	// Normalizar JID (adiciona @s.whatsapp.net se for apenas número)
	normalizedTo := normalizeJID(to)

	// Extrair duração (seconds)
	secondsStr := c.PostForm("seconds")
	seconds, _ := strconv.Atoi(secondsStr)

	// Extrair flag PTT explícita
	pttStr := c.PostForm("ptt")
	ptt := pttStr == "true" || pttStr == "1"

	mediaType := file.Header.Get("Content-Type")

	msg, err := h.service.Send(c.Request.Context(), messageSvc.SendInput{
		InstanceID: instanceID,
		To:         normalizedTo,
		Type:       "audio",
		MediaData:  fileData,
		MediaType:  mediaType,
		Seconds:    seconds,
		PTT:        ptt,
	})
	if err != nil {
		if errors.Is(err, messageSvc.ErrInstanceNotConnected) {
			response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		} else {
			response.Error(c, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(c, http.StatusOK, msg)
}

func (h *MessageHandler) sendDocument(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return
	}
	to := c.PostForm("to")
	fileName := c.PostForm("filename")
	caption := c.PostForm("caption")

	if to == "" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "campo 'to' é obrigatório")
		return
	}

	// Obter arquivo
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "arquivo não fornecido")
		return
	}

	// Usar nome do arquivo enviado se não fornecido
	if fileName == "" {
		fileName = file.Filename
	}

	// Abrir arquivo
	src, err := file.Open()
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao abrir arquivo")
		return
	}
	defer src.Close()

	// Ler dados do arquivo
	fileData, err := io.ReadAll(src)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao ler arquivo")
		return
	}

	// Normalizar JID (adiciona @s.whatsapp.net se for apenas número)
	normalizedTo := normalizeJID(to)

	msg, err := h.service.Send(c.Request.Context(), messageSvc.SendInput{
		InstanceID: instanceID,
		To:         normalizedTo,
		Type:       "document",
		MediaData:  fileData,
		MediaType:  file.Header.Get("Content-Type"),
		FileName:   fileName,
		Caption:    caption,
	})
	if err != nil {
		if errors.Is(err, messageSvc.ErrInstanceNotConnected) {
			response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		} else {
			response.Error(c, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(c, http.StatusOK, msg)
}

func (h *MessageHandler) list(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return
	}
	list, err := h.service.List(c.Request.Context(), instanceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, http.StatusOK, list)
}
