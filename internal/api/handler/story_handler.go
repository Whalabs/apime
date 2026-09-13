package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/response"
	messageSvc "github.com/open-apime/apime/internal/service/message"
)

// statusBroadcastJID is the fixed destination for stories. WhatsApp resolves the audience from the
// account's status privacy settings, so the caller never lists recipients.
const statusBroadcastJID = "status@broadcast"

// storyMediaTypes are the media kinds a story accepts. Documents and stickers are not postable as
// status, so they are rejected here instead of failing further down.
var storyMediaTypes = map[string]bool{
	"image": true,
	"video": true,
}

type sendStoryTextRequest struct {
	Text string `json:"text" binding:"required"`
}

// sendStoryText posts a text story. The audience comes from the account's status privacy, readable
// via GET /whatsapp/status-privacy.
func (h *MessageHandler) sendStoryText(c *gin.Context) {
	instanceID, ok := requireInstanceToken(c)
	if !ok {
		return
	}
	var req sendStoryTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	msg, err := h.service.Send(c.Request.Context(), messageSvc.SendInput{
		InstanceID: instanceID,
		To:         statusBroadcastJID,
		Type:       "text",
		Text:       req.Text,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, http.StatusOK, msg)
}

// sendStoryMedia posts an image or video story.
func (h *MessageHandler) sendStoryMedia(c *gin.Context) {
	instanceID, ok := requireInstanceToken(c)
	if !ok {
		return
	}
	mediaType := c.PostForm("type")
	if !storyMediaTypes[mediaType] {
		response.ErrorWithMessage(c, http.StatusBadRequest, "tipo deve ser 'image' ou 'video'")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "arquivo não fornecido")
		return
	}
	src, err := file.Open()
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao abrir arquivo")
		return
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao ler arquivo")
		return
	}

	msg, err := h.service.Send(c.Request.Context(), messageSvc.SendInput{
		InstanceID: instanceID,
		To:         statusBroadcastJID,
		Type:       mediaType,
		MediaData:  fileData,
		MediaType:  file.Header.Get("Content-Type"),
		Caption:    c.PostForm("caption"),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, http.StatusOK, msg)
}

// requireInstanceToken mirrors the guard the whatsapp handler uses: these routes act on one
// instance, so a user-wide JWT must not reach them.
func requireInstanceToken(c *gin.Context) (string, bool) {
	instanceID := c.Param("id")
	if c.GetString("authType") != "instance_token" {
		response.ErrorWithMessage(c, http.StatusForbidden, "endpoint disponível apenas com token de instância")
		return "", false
	}
	if c.GetString("instanceID") != instanceID {
		response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
		return "", false
	}
	return instanceID, true
}
