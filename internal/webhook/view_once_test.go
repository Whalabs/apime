package webhook

import (
	"context"
	"testing"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func undecryptable(fromMe bool, kind events.UnavailableType) *events.UndecryptableMessage {
	me := types.NewJID("5511999990000", types.DefaultUserServer)
	contact := types.NewJID("5511988887777", types.DefaultUserServer)
	src := types.MessageSource{Chat: types.NewJID("123456789", types.HiddenUserServer), Sender: contact, IsFromMe: fromMe}
	if fromMe {
		src.Sender = me
		src.SenderAlt = me
		src.RecipientAlt = contact
	} else {
		src.SenderAlt = contact
	}
	return &events.UndecryptableMessage{Info: types.MessageInfo{MessageSource: src, ID: "ABC"}, UnavailableType: kind}
}

func TestOwnViewOnceReachesTheContactThread(t *testing.T) {
	// A view-once sent from the phone used to be dropped, so it never showed up in the panel.
	got := (&EventHandler{}).normalizeEvent(context.Background(), "inst", nil, undecryptable(true, events.UnavailableTypeViewOnce))
	if got["type"] != "message" || got["isFromMe"] != true || got["unavailableType"] != "view_once" {
		t.Fatalf("unexpected payload: %v", got)
	}
	// SenderAlt is our own number here; the thread must be the recipient's.
	if got["chatJID"] != "5511988887777@s.whatsapp.net" {
		t.Errorf("chatJID = %v, want the contact", got["chatJID"])
	}
}

func TestOwnUndecryptableStillIgnored(t *testing.T) {
	got := (&EventHandler{}).normalizeEvent(context.Background(), "inst", nil, undecryptable(true, events.UnavailableTypeUnknown))
	if got["type"] != "ignore" {
		t.Fatalf("type = %v, want ignore", got["type"])
	}
}

func TestInboundViewOnceUnchanged(t *testing.T) {
	got := (&EventHandler{}).normalizeEvent(context.Background(), "inst", nil, undecryptable(false, events.UnavailableTypeViewOnce))
	if got["isFromMe"] != false || got["chatJID"] != "5511988887777@s.whatsapp.net" {
		t.Fatalf("unexpected payload: %v", got)
	}
}
