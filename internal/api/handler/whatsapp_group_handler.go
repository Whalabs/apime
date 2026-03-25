package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/open-apime/apime/internal/pkg/response"
)

type createGroupRequest struct {
	Name         string   `json:"name" binding:"required"`
	Participants []string `json:"participants"`
}

func (h *WhatsAppHandler) createGroup(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	participants := make([]types.JID, 0, len(req.Participants))
	for _, p := range req.Participants {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, "@") {
			p = strings.TrimPrefix(p, "+") + "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(p)
		if err != nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "participant inválido")
			return
		}
		participants = append(participants, jid)
	}

	info, err := client.CreateGroup(c.Request.Context(), whatsmeow.ReqCreateGroup{
		Name:         strings.TrimSpace(req.Name),
		Participants: participants,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, info)
}

type updateGroupParticipantsRequest struct {
	Action       string   `json:"action" binding:"required"` // add/remove/promote/demote
	Participants []string `json:"participants" binding:"required"`
}

func (h *WhatsAppHandler) updateGroupParticipants(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groupStr := strings.TrimSpace(c.Param("group"))
	if !strings.Contains(groupStr, "@") {
		groupStr = groupStr + "@g.us"
	}
	groupJID, err := types.ParseJID(groupStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "group inválido")
		return
	}

	var req updateGroupParticipantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	var action whatsmeow.ParticipantChange
	switch strings.ToLower(strings.TrimSpace(req.Action)) {
	case "add":
		action = whatsmeow.ParticipantChangeAdd
	case "remove":
		action = whatsmeow.ParticipantChangeRemove
	case "promote":
		action = whatsmeow.ParticipantChangePromote
	case "demote":
		action = whatsmeow.ParticipantChangeDemote
	default:
		response.ErrorWithMessage(c, http.StatusBadRequest, "action inválida")
		return
	}

	participants := make([]types.JID, 0, len(req.Participants))
	for _, p := range req.Participants {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, "@") {
			p = strings.TrimPrefix(p, "+") + "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(p)
		if err != nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "participant inválido")
			return
		}
		participants = append(participants, jid)
	}

	res, err := client.UpdateGroupParticipants(c.Request.Context(), groupJID, participants, action)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"participants": res})
}

func (h *WhatsAppHandler) listGroupJoinRequests(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groupStr := strings.TrimSpace(c.Param("group"))
	if !strings.Contains(groupStr, "@") {
		groupStr = groupStr + "@g.us"
	}
	groupJID, err := types.ParseJID(groupStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "group inválido")
		return
	}

	list, err := client.GetGroupRequestParticipants(c.Request.Context(), groupJID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"requests": list})
}

type updateGroupJoinRequestsRequest struct {
	Action       string   `json:"action" binding:"required"` // approve/reject
	Participants []string `json:"participants" binding:"required"`
}

func (h *WhatsAppHandler) updateGroupJoinRequests(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groupStr := strings.TrimSpace(c.Param("group"))
	if !strings.Contains(groupStr, "@") {
		groupStr = groupStr + "@g.us"
	}
	groupJID, err := types.ParseJID(groupStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "group inválido")
		return
	}

	var req updateGroupJoinRequestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	var action whatsmeow.ParticipantRequestChange
	switch strings.ToLower(strings.TrimSpace(req.Action)) {
	case "approve":
		action = whatsmeow.ParticipantChangeApprove
	case "reject":
		action = whatsmeow.ParticipantChangeReject
	default:
		response.ErrorWithMessage(c, http.StatusBadRequest, "action inválida")
		return
	}

	participants := make([]types.JID, 0, len(req.Participants))
	for _, p := range req.Participants {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, "@") {
			p = strings.TrimPrefix(p, "+") + "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(p)
		if err != nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "participant inválido")
			return
		}
		participants = append(participants, jid)
	}

	res, err := client.UpdateGroupRequestParticipants(c.Request.Context(), groupJID, participants, action)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"participants": res})
}

func (h *WhatsAppHandler) getGroupInfo(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groupStr := strings.TrimSpace(c.Param("group"))
	if !strings.Contains(groupStr, "@") {
		groupStr = groupStr + "@g.us"
	}
	groupJID, err := types.ParseJID(groupStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "group inválido")
		return
	}

	info, err := client.GetGroupInfo(c.Request.Context(), groupJID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, http.StatusOK, info)
}

func (h *WhatsAppHandler) getGroupInviteLink(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groupStr := strings.TrimSpace(c.Param("group"))
	if !strings.Contains(groupStr, "@") {
		groupStr = groupStr + "@g.us"
	}
	groupJID, err := types.ParseJID(groupStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "group inválido")
		return
	}

	reset := strings.EqualFold(strings.TrimSpace(c.Query("reset")), "true")
	link, err := client.GetGroupInviteLink(c.Request.Context(), groupJID, reset)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"link": link})
}

type getGroupInfoFromLinkRequest struct {
	Link string `json:"link" binding:"required"`
}

func (h *WhatsAppHandler) getGroupInfoFromLink(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	var req getGroupInfoFromLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	info, err := client.GetGroupInfoFromLink(c.Request.Context(), strings.TrimSpace(req.Link))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, info)
}

type joinGroupWithLinkRequest struct {
	Link string `json:"link" binding:"required"`
}

func (h *WhatsAppHandler) joinGroupWithLink(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	var req joinGroupWithLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	jid, err := client.JoinGroupWithLink(c.Request.Context(), strings.TrimSpace(req.Link))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"group": jid.String()})
}

func (h *WhatsAppHandler) leaveGroup(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groupStr := strings.TrimSpace(c.Param("group"))
	if !strings.Contains(groupStr, "@") {
		groupStr = groupStr + "@g.us"
	}
	groupJID, err := types.ParseJID(groupStr)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "group inválido")
		return
	}

	if err := client.LeaveGroup(c.Request.Context(), groupJID); err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *WhatsAppHandler) listGroups(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}

	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}

	groups, err := client.GetJoinedGroups(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, http.StatusOK, groups)
}
