package webhook

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// broadcastRecipients flattens the recipients of a broadcast list message into the payload shape.
// WhatsApp emits a single event for the whole list, so this array is the only place the consumer
// learns who actually received a message WE sent. It is empty for an inbound broadcast: whatsmeow
// fills the array only when the message is ours, and there the sender plus broadcastListOwner are
// what identify the conversation.
//
// Both identities go out when known: the PN is what most consumers key conversations by, and the
// LID is the stable identity that survives a number change. Entries with neither are dropped rather
// than emitted empty, so the consumer can trust every entry is addressable.
//
// A list sent from a companion (not the phone) echoes to the other devices with the LID alone, and a
// consumer that has never seen that LID opens a nameless duplicate contact. resolvePN fills the PN
// from the local LID map in that case; nil skips it.
func broadcastRecipients(recipients []types.BroadcastRecipient, resolvePN func(types.JID) types.JID) []map[string]string {
	if len(recipients) == 0 {
		return nil
	}
	out := make([]map[string]string, 0, len(recipients))
	for _, r := range recipients {
		entry := map[string]string{}
		if r.PN.IsEmpty() && !r.LID.IsEmpty() && resolvePN != nil {
			r.PN = resolvePN(r.LID)
		}
		if !r.PN.IsEmpty() {
			entry["pn"] = r.PN.String()
		}
		if !r.LID.IsEmpty() {
			entry["lid"] = r.LID.String()
		}
		if len(entry) == 0 {
			continue
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// storedListRecipients reads a list's members from the session store, syncing from the server once
// when the list is unknown. A list message another of our devices sent reaches us through a retry,
// encrypted for this device alone, so it carries no recipients; without them the consumer cannot
// place the copies in each recipient's conversation.
func storedListRecipients(ctx context.Context, client *whatsmeow.Client, list types.JID) []types.BroadcastRecipient {
	if client == nil {
		return nil
	}
	info, err := client.GetBroadcastListInfo(ctx, list)
	if err != nil {
		if syncErr := client.SyncBroadcastLists(ctx); syncErr != nil {
			return nil
		}
		if info, err = client.GetBroadcastListInfo(ctx, list); err != nil {
			return nil
		}
	}
	out := make([]types.BroadcastRecipient, 0, len(info.Participants))
	for _, p := range info.Participants {
		out = append(out, types.BroadcastRecipient{LID: p.LID, PN: p.PN})
	}
	return out
}
