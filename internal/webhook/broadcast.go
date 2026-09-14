package webhook

import "go.mau.fi/whatsmeow/types"

// broadcastRecipients flattens the recipients of a broadcast list message into the payload shape.
// WhatsApp emits a single event for the whole list, so this array is the only place the consumer
// learns who actually received a message WE sent. It is empty for an inbound broadcast: whatsmeow
// fills the array only when the message is ours, and there the sender plus broadcastListOwner are
// what identify the conversation.
//
// Both identities go out when known: the PN is what most consumers key conversations by, and the
// LID is the stable identity that survives a number change. Entries with neither are dropped rather
// than emitted empty, so the consumer can trust every entry is addressable.
func broadcastRecipients(recipients []types.BroadcastRecipient) []map[string]string {
	if len(recipients) == 0 {
		return nil
	}
	out := make([]map[string]string, 0, len(recipients))
	for _, r := range recipients {
		entry := map[string]string{}
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
