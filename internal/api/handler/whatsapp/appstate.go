package whatsapp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/response"
)

type resyncAppStateRequest struct {
	// Collections to resync. Empty means the two that break user-visible features when stale:
	// critical_unblock_low (the contact list, which is the audience of a status post) and regular
	// (which carries the broadcast lists).
	Collections []string `json:"collections"`
}

// resyncAppState discards the local copy of the given app state collections and downloads a fresh
// snapshot. It repairs a collection that diverged from the server ("mismatching LTHash"), which
// whatsmeow reports once and then stops applying patches for, with no retry of its own.
func (h *Handler) resyncAppState(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	resyncer, ok := h.sessionManager.(appStateResyncer)
	if !ok {
		response.ErrorWithMessage(c, http.StatusNotImplemented, "ressincronização não disponível")
		return
	}
	var req resyncAppStateRequest
	// An empty body is the common case (resync the defaults), so a bind failure only matters when
	// something was actually sent.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, err)
			return
		}
	}
	resynced, skipped, err := resyncer.ResyncAppState(c.Request.Context(), instanceID, req.Collections)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	// "skipped" travels so an empty "resynced" is not read as a failure: a collection resynced
	// minutes ago is barred by the cooldown, which is a normal outcome.
	response.Success(c, http.StatusOK, gin.H{"resynced": resynced, "skipped": skipped})
}
