package message

import (
	"context"
	"errors"
	"testing"

	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"

	"github.com/open-apime/apime/internal/storage"
	"github.com/open-apime/apime/internal/storage/model"
)

func TestValidWhatsAppID(t *testing.T) {
	valid := []string{"3EB0A1B2C3D4E5F60718", "3EB0ABCDEF0123456789ABCD", "3A0123456789ABCDEF0123456789ABCD"}
	for _, id := range valid {
		if !ValidWhatsAppID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}
	invalid := []string{"", "3eb0a1b2c3d4e5f60718", "3EB0", "3EB0A1B2C3D4E5G60718", "3EB0-A1B2-C3D4-E5F6",
		"3EB0A1B2C3D4E5F60718 ", "0123456789ABCDEF0123456789ABCDEF0"}
	for _, id := range invalid {
		if ValidWhatsAppID(id) {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}

func TestResolveSendIDGeneratesOnceAndKeepsCallerID(t *testing.T) {
	calls := 0
	gen := func() types.MessageID { calls++; return "3EB0GENERATED" }

	if got := resolveSendID("3EB0CALLER0000000000", gen); got != "3EB0CALLER0000000000" || calls != 0 {
		t.Fatalf("caller id: got %q, generator calls %d", got, calls)
	}
	if got := resolveSendID("", gen); got != "3EB0GENERATED" || calls != 1 {
		t.Fatalf("generated id: got %q, generator calls %d", got, calls)
	}
}

type fakeMessageRepo struct {
	storage.MessageRepository
	byWhatsAppID map[string]model.Message
}

// Mirrors the store filter: same instance and a status WhatsApp accepted.
func (f *fakeMessageRepo) GetByWhatsAppIDForInstance(_ context.Context, instanceID, id string) (model.Message, error) {
	m, ok := f.byWhatsAppID[id]
	if !ok || m.InstanceID != instanceID || m.Status == "queued" || m.Status == "sending" || m.Status == "failed" {
		return model.Message{}, errors.New("not found")
	}
	return m, nil
}

func TestAlreadySentReplaysOnlyForTheSameInstance(t *testing.T) {
	sent := model.Message{ID: "row-1", InstanceID: "inst-a", WhatsAppID: "3EB0A1B2C3D4E5F60718", Status: "sent"}
	s := &Service{repo: &fakeMessageRepo{byWhatsAppID: map[string]model.Message{sent.WhatsAppID: sent}}, log: zap.NewNop()}

	if got, ok := s.alreadySent(context.Background(), SendInput{InstanceID: "inst-a", WhatsAppID: sent.WhatsAppID}); !ok || got.ID != "row-1" {
		t.Fatalf("same instance should replay, got %+v ok=%v", got, ok)
	}
	if _, ok := s.alreadySent(context.Background(), SendInput{InstanceID: "inst-b", WhatsAppID: sent.WhatsAppID}); ok {
		t.Fatal("another instance must not replay")
	}
	if _, ok := s.alreadySent(context.Background(), SendInput{InstanceID: "inst-a", WhatsAppID: "3EB0FFFFFFFFFFFFFFFF"}); ok {
		t.Fatal("unknown id must not replay")
	}
	if _, ok := s.alreadySent(context.Background(), SendInput{InstanceID: "inst-a"}); ok {
		t.Fatal("no caller id must not replay")
	}
}
