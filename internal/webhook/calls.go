package webhook

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/open-apime/apime/internal/voip"
)

// normalizeCall turns call events into call.* webhooks. It returns before the generic
// "raw" dump on purpose: the offer node carries the encrypted call key.
func (h *EventHandler) normalizeCall(ctx context.Context, client *whatsmeow.Client, evt any) (map[string]interface{}, bool) {
	switch evt := evt.(type) {
	case *events.CallOffer:
		result := h.callMeta(ctx, client, evt.BasicCallMeta)
		// "Call ended" notices (e.g. answered on another device) arrive offer-shaped.
		isVideo := false
		if evt.Data != nil {
			ag := evt.Data.AttrGetter()
			if ag.OptionalString("is_call_ended") == "1" || ag.OptionalString("terminate_reason") != "" {
				result["type"] = "call.ended"
				result["reason"] = ag.OptionalString("terminate_reason")
				return result, true
			}
			_, isVideo = evt.Data.GetOptionalChildByTag("video")
		}
		result["type"] = "call.offer"
		result["isVideo"] = isVideo
		result["platform"] = evt.RemotePlatform
		return result, true
	case *events.CallTerminate:
		result := h.callMeta(ctx, client, evt.BasicCallMeta)
		result["type"] = "call.ended"
		result["reason"] = evt.Reason
		return result, true
	case *events.CallReject:
		result := h.callMeta(ctx, client, evt.BasicCallMeta)
		result["type"] = "call.ended"
		result["reason"] = "rejected"
		return result, true
	case *voip.StateEvent:
		result := map[string]interface{}{
			"type":      "call.state",
			"callId":    evt.ID,
			"from":      evt.Peer,
			"direction": evt.Direction,
			"state":     evt.State,
			"startedAt": evt.StartedAt,
		}
		if peer, err := types.ParseJID(evt.Peer); err == nil {
			if pn := h.resolvePNFromStore(ctx, client, peer); !pn.IsEmpty() {
				result["from"] = pn.String()
				result["lid"] = peer.ToNonAD().String()
			}
		}
		if evt.Reason != "" {
			result["reason"] = evt.Reason
		}
		return result, true
	}
	return nil, false
}

// callMeta resolves the caller to a phone JID when possible, keeping the LID alongside,
// the same pair message webhooks carry.
func (h *EventHandler) callMeta(ctx context.Context, client *whatsmeow.Client, meta types.BasicCallMeta) map[string]interface{} {
	caller := meta.CallCreator
	if caller.IsEmpty() {
		caller = meta.From
	}
	result := map[string]interface{}{
		"callId":    meta.CallID,
		"from":      caller.ToNonAD().String(),
		"timestamp": meta.Timestamp,
	}
	if caller.Server == types.HiddenUserServer {
		result["lid"] = caller.ToNonAD().String()
		pn := meta.CallCreatorAlt
		if pn.Server != types.DefaultUserServer {
			pn = h.resolvePNFromStore(ctx, client, caller)
		}
		if !pn.IsEmpty() {
			result["from"] = pn.ToNonAD().String()
		}
	}
	// Needed by POST /calls/:callId/reject on instances without the engine.
	result["rejectFrom"] = meta.From.String()
	return result
}
