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
	// Styling of the status card. Omitted means WhatsApp green on white with the system font,
	// which is what the official composer offers first. Colours are opaque ARGB.
	BackgroundArgb uint32 `json:"backgroundArgb"`
	TextArgb       uint32 `json:"textArgb"`
	Font           string `json:"font"`
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
		InstanceID:     instanceID,
		To:             statusBroadcastJID,
		Type:           "text",
		Text:           req.Text,
		BackgroundArgb: req.BackgroundArgb,
		TextArgb:       req.TextArgb,
		Font:           req.Font,
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
