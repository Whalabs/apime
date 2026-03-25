package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types"

	"github.com/open-apime/apime/internal/pkg/response"
)

func (h *WhatsAppHandler) newsletterSubscribeLiveUpdates(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	jidStr := strings.TrimSpace(c.Param("jid"))
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "jid inválido")
		return
	}
	dur, err := client.NewsletterSubscribeLiveUpdates(c.Request.Context(), jid)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"duration_seconds": int64(dur.Seconds())})
}

type newsletterMarkViewedRequest struct {
	ServerIDs []string `json:"server_ids" binding:"required"`
}

func (h *WhatsAppHandler) newsletterMarkViewed(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	jidStr := strings.TrimSpace(c.Param("jid"))
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "jid inválido")
		return
	}
	var req newsletterMarkViewedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	ids := make([]types.MessageServerID, 0, len(req.ServerIDs))
	for _, s := range req.ServerIDs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		idInt, err := strconv.Atoi(s)
		if err != nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "server_id inválido")
			return
		}
		ids = append(ids, types.MessageServerID(idInt))
	}
	if err := client.NewsletterMarkViewed(c.Request.Context(), jid, ids); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"status": "ok"})
}

type newsletterSendReactionRequest struct {
	ServerID  string `json:"server_id" binding:"required"`
	Reaction  string `json:"reaction"`
	MessageID string `json:"message_id"`
}

func (h *WhatsAppHandler) newsletterSendReaction(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	jidStr := strings.TrimSpace(c.Param("jid"))
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "jid inválido")
		return
	}
	var req newsletterSendReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	serverIDStr := strings.TrimSpace(req.ServerID)
	if serverIDStr == "" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "server_id inválido")
		return
	}
	serverIDInt, err := strconv.Atoi(serverIDStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "server_id inválido")
		return
	}
	if err := client.NewsletterSendReaction(
		c.Request.Context(),
		jid,
		types.MessageServerID(serverIDInt),
		strings.TrimSpace(req.Reaction),
		types.MessageID(strings.TrimSpace(req.MessageID)),
	); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *WhatsAppHandler) getNewsletterMessageUpdates(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	jidStr := strings.TrimSpace(c.Param("jid"))
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "jid inválido")
		return
	}
	updates, err := client.GetNewsletterMessageUpdates(c.Request.Context(), jid, nil)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"updates": updates})
}
