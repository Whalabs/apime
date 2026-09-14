package whatsapp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/open-apime/apime/internal/pkg/response"
)

// broadcastListResponse is the API shape of a broadcast list. Participants carry both identities
// because the consumer keys conversations by phone number but must survive a number change.
type broadcastListResponse struct {
	JID          string                      `json:"jid"`
	Name         string                      `json:"name"`
	Participants []broadcastParticipantEntry `json:"participants"`
	LabelIDs     []string                    `json:"labelIds"`
	UpdatedAt    string                      `json:"updatedAt"`
}

type broadcastParticipantEntry struct {
	PN  string `json:"pn,omitempty"`
	LID string `json:"lid,omitempty"`
}

func toBroadcastListResponse(info types.BroadcastListInfo) broadcastListResponse {
	participants := make([]broadcastParticipantEntry, 0, len(info.Participants))
	for _, p := range info.Participants {
		entry := broadcastParticipantEntry{}
		if !p.PN.IsEmpty() {
			entry.PN = p.PN.String()
		}
		if !p.LID.IsEmpty() {
			entry.LID = p.LID.String()
		}
		participants = append(participants, entry)
	}
	labelIDs := info.LabelIDs
	if labelIDs == nil {
		labelIDs = []string{}
	}
	return broadcastListResponse{
		JID:          info.JID.String(),
		Name:         info.Name,
		Participants: participants,
		LabelIDs:     labelIDs,
		UpdatedAt:    info.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (h *Handler) listBroadcastLists(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	lists, err := client.GetBroadcastLists(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]broadcastListResponse, 0, len(lists))
	for _, list := range lists {
		out = append(out, toBroadcastListResponse(list))
	}
	response.Success(c, http.StatusOK, gin.H{"lists": out})
}

func (h *Handler) getBroadcastList(c *gin.Context) {
	instanceID, ok := h.requireInstanceToken(c)
	if !ok {
		return
	}
	client, err := h.sessionManager.GetClient(instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		return
	}
	jid, err := parseBroadcastListJID(c.Param("jid"))
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "jid de lista inválido")
		return
	}
	info, err := client.GetBroadcastListInfo(c.Request.Context(), jid)
	// A list that never synced, or one whose participants did not resolve, is a legitimate state of
	// the account rather than a server fault, so neither is a 500.
	if errors.Is(err, whatsmeow.ErrBroadcastListNotFound) || errors.Is(err, whatsmeow.ErrBroadcastListEmpty) {
		response.ErrorWithMessage(c, http.StatusNotFound, "lista não encontrada, ela só aparece após a sincronização com o celular")
		return
	} else if errors.Is(err, whatsmeow.ErrBroadcastListUnsupported) {
		response.ErrorWithMessage(c, http.StatusServiceUnavailable, "listas de transmissão indisponíveis nesta instância")
		return
	} else if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, http.StatusOK, toBroadcastListResponse(*info))
}

// parseBroadcastListJID accepts both the bare list id and the full JID, since the id alone is what
// shows up in a webhook chatJID for callers that strip the server.
func parseBroadcastListJID(raw string) (types.JID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.EmptyJID, errors.New("jid vazio")
	}
	if !strings.Contains(raw, "@") {
		raw += "@" + types.BroadcastServer
	}
	jid, err := types.ParseJID(raw)
	if err != nil {
		return types.EmptyJID, err
	}
	if jid.Server != types.BroadcastServer {
		return types.EmptyJID, errors.New("jid não é de lista de transmissão")
	}
	return jid, nil
}
